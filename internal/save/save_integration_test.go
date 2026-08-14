package save

import (
	"os"
	"path/filepath"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadIntegration(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)

	// テスト用エンティティを作成
	_, pErr := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, pErr)

	_, npcErr := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 8, Y: 8}, "bat")
	require.NoError(t, npcErr)

	// セーブマネージャーを作成
	saveManager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)

	// セーブテスト
	err = saveManager.SaveWorld(world, "test_slot")
	require.NoError(t, err)

	// セーブファイルの存在確認
	saveFile := filepath.Join(testDir, "test_slot.json")
	_, err = os.Stat(saveFile)
	require.NoError(t, err, "Save file should exist")

	// 新しいワールドを作成
	newWorld := testutil.InitTestWorld(t)

	// ロードテスト
	err = saveManager.LoadWorld(newWorld, "test_slot")
	require.NoError(t, err)

	// データの検証
	playerCount := 0
	npcCount := 0

	playerQuery := ecs.NewFilter1[gc.Player](newWorld.ECS).Query()
	for playerQuery.Next() {
		playerCount++
	}

	npcQuery := ecs.NewFilter1[gc.FactionEnemy](newWorld.ECS).Query()
	for npcQuery.Next() {
		npcCount++
	}

	assert.Equal(t, 1, playerCount, "プレイヤーが1個存在する")
	assert.Equal(t, 1, npcCount, "丸ごと保存のためNPCも保存・復元される")
}

func TestSaveSlotInfo(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	// セーブマネージャーを作成
	saveManager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)

	// 初期状態（セーブファイルなし）でセーブファイルの存在を確認
	slotFile := filepath.Join(testDir, "slot1.json")
	_, err = os.Stat(slotFile)
	require.Error(t, err, "Save file should not exist initially")

	// 1つのセーブファイルを作成
	err = saveManager.SaveWorld(world, "slot1")
	require.NoError(t, err)

	// セーブファイル作成後の状態を確認
	_, err = os.Stat(slotFile)
	require.NoError(t, err, "Save file should exist after save")

	// 複数のスロットにセーブ
	err = saveManager.SaveWorld(world, "slot2")
	require.NoError(t, err)
	err = saveManager.SaveWorld(world, "slot3")
	require.NoError(t, err)

	// 全てのスロットファイルが存在することを確認
	slot2File := filepath.Join(testDir, "slot2.json")
	slot3File := filepath.Join(testDir, "slot3.json")

	_, err = os.Stat(slot2File)
	require.NoError(t, err, "Slot 2 save file should exist")
	_, err = os.Stat(slot3File)
	require.NoError(t, err, "Slot 3 save file should exist")

	t.Logf("All save files created successfully")
}

// TestSaveLoadInPlace は同一ワールドに対してsave→loadするケースを検証する。
// ゲーム内ではロード時に既存のworldをclearWorldしてから復元するため、
// シングルトンコンポーネントが正しく保持されることを確認する
func TestSaveLoadInPlace(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, pErr := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, pErr)

	// GameProgressにデータを設定
	query.GetGameProgress(world).MarkDungeonCleared("遺跡")

	sm, err := NewSerializationManager(WithSaveDir(tempDir))
	require.NoError(t, err)
	err = sm.SaveWorld(world, "inplace")
	require.NoError(t, err)

	// 同一ワールドにロードする（ゲーム内と同じフロー）
	err = sm.LoadWorld(world, "inplace")
	require.NoError(t, err)

	// シングルトンのGameProgressがパニックせずアクセスできることを確認
	gp := query.GetGameProgress(world)
	require.NotNil(t, gp, "GameProgressがnilであってはならない")
	assert.True(t, gp.IsDungeonCleared("遺跡"))

	// Dungeonは丸ごと保存で復元されるのでnilにならない
	d := query.GetDungeon(world)
	assert.NotNil(t, d, "Dungeonが存在する")
}

// TestSaveLoadGameProgress はGameProgressのセーブ・ロードを検証する
func TestSaveLoadGameProgress(t *testing.T) {
	t.Parallel()

	t.Run("ダンジョンクリアフラグの保存と復元", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成（セーブ対象のエンティティが必要）
		_, pErr := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, pErr)

		// ダンジョンクリアフラグを設定
		query.GetGameProgress(world).MarkDungeonCleared("遺跡")
		query.GetGameProgress(world).MarkDungeonCleared("洞窟")

		// JSON生成→復元のラウンドトリップ
		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		err = sm.RestoreWorldFromJSON(newWorld, jsonStr)
		require.NoError(t, err)

		// 復元後のGameProgressを検証
		assert.True(t, query.GetGameProgress(newWorld).IsDungeonCleared("遺跡"))
		assert.True(t, query.GetGameProgress(newWorld).IsDungeonCleared("洞窟"))
		assert.False(t, query.GetGameProgress(newWorld).IsDungeonCleared("森林"))
	})

	t.Run("イベント状態の保存と復元", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, pErr := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, pErr)

		// イベント状態を設定
		query.GetGameProgress(world).SetEventActive("all_cleared")
		query.GetGameProgress(world).MarkEventSeen("all_cleared")

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		err = sm.RestoreWorldFromJSON(newWorld, jsonStr)
		require.NoError(t, err)

		// 視聴済みイベントはIsEventUnseenがfalseになる
		assert.False(t, query.GetGameProgress(newWorld).IsEventUnseen("all_cleared"))
		ev := query.GetGameProgress(newWorld).Events["all_cleared"]
		assert.True(t, ev.Active)
		assert.True(t, ev.Seen)
	})

	t.Run("空のGameProgressの保存と復元", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, pErr := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, pErr)

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		err = sm.RestoreWorldFromJSON(newWorld, jsonStr)
		require.NoError(t, err)

		assert.Empty(t, query.GetGameProgress(newWorld).ClearedDungeons)
		assert.Empty(t, query.GetGameProgress(newWorld).Events)
	})
}
