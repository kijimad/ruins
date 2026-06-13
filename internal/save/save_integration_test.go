package save

import (
	"os"
	"path/filepath"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestSaveLoadIntegration(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)

	// テスト用エンティティを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})

	npc := world.Manager.NewEntity()
	npc.AddComponent(world.Components.Name, &gc.Name{Name: "テストNPC"})
	npc.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemyData{})

	// セーブマネージャーを作成
	saveManager := NewSerializationManager(testDir)

	// セーブテスト
	err := saveManager.SaveWorld(world, "test_slot")
	require.NoError(t, err)

	// セーブファイルの存在確認
	saveFile := filepath.Join(testDir, "test_slot.json")
	_, err = os.Stat(saveFile)
	assert.NoError(t, err, "Save file should exist")

	// 新しいワールドを作成
	newWorld := testutil.InitTestWorld(t)

	// ロードテスト
	err = saveManager.LoadWorld(newWorld, "test_slot")
	require.NoError(t, err)

	// データの検証
	playerCount := 0
	npcCount := 0

	newWorld.Manager.Join(newWorld.Components.Player).Visit(ecs.Visit(func(_ ecs.Entity) {
		playerCount++
	}))

	newWorld.Manager.Join(newWorld.Components.FactionEnemy).Visit(ecs.Visit(func(_ ecs.Entity) {
		npcCount++
	}))

	assert.Equal(t, 1, playerCount, "プレイヤーが1個存在する")
	assert.Equal(t, 0, npcCount, "NPCは保存されない（プレイヤーとアイテムのみ保存）")
}

func TestSaveSlotInfo(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	// セーブマネージャーを作成
	saveManager := NewSerializationManager(testDir)

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)

	// 初期状態（セーブファイルなし）でセーブファイルの存在を確認
	slotFile := filepath.Join(testDir, "slot1.json")
	_, err := os.Stat(slotFile)
	assert.Error(t, err, "Save file should not exist initially")

	// 1つのセーブファイルを作成
	err = saveManager.SaveWorld(world, "slot1")
	require.NoError(t, err)

	// セーブファイル作成後の状態を確認
	_, err = os.Stat(slotFile)
	assert.NoError(t, err, "Save file should exist after save")

	// 複数のスロットにセーブ
	err = saveManager.SaveWorld(world, "slot2")
	require.NoError(t, err)
	err = saveManager.SaveWorld(world, "slot3")
	require.NoError(t, err)

	// 全てのスロットファイルが存在することを確認
	slot2File := filepath.Join(testDir, "slot2.json")
	slot3File := filepath.Join(testDir, "slot3.json")

	_, err = os.Stat(slot2File)
	assert.NoError(t, err, "Slot 2 save file should exist")
	_, err = os.Stat(slot3File)
	assert.NoError(t, err, "Slot 3 save file should exist")

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
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})

	// GameProgressにデータを設定
	worldhelper.GetGameProgress(world).MarkDungeonCleared("遺跡")

	sm := NewSerializationManager(tempDir)
	err := sm.SaveWorld(world, "inplace")
	require.NoError(t, err)

	// 同一ワールドにロードする（ゲーム内と同じフロー）
	err = sm.LoadWorld(world, "inplace")
	require.NoError(t, err)

	// シングルトンのGameProgressがパニックせずアクセスできることを確認
	gp := worldhelper.GetGameProgress(world)
	require.NotNil(t, gp, "GameProgressがnilであってはならない")
	assert.True(t, gp.IsDungeonCleared("遺跡"))

	// DungeonはInitSingletonで再作成されるのでnilにならない
	d := worldhelper.GetDungeon(world)
	assert.NotNil(t, d, "Dungeonが存在する")
}

// TestSaveLoadGameProgress はGameProgressのセーブ・ロードを検証する
func TestSaveLoadGameProgress(t *testing.T) {
	t.Parallel()

	t.Run("ダンジョンクリアフラグの保存と復元", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成（セーブ対象のエンティティが必要）
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})

		// ダンジョンクリアフラグを設定
		worldhelper.GetGameProgress(world).MarkDungeonCleared("遺跡")
		worldhelper.GetGameProgress(world).MarkDungeonCleared("洞窟")

		// JSON生成→復元のラウンドトリップ
		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		err = sm.RestoreWorldFromJSON(newWorld, jsonStr)
		require.NoError(t, err)

		// 復元後のGameProgressを検証
		assert.True(t, worldhelper.GetGameProgress(newWorld).IsDungeonCleared("遺跡"))
		assert.True(t, worldhelper.GetGameProgress(newWorld).IsDungeonCleared("洞窟"))
		assert.False(t, worldhelper.GetGameProgress(newWorld).IsDungeonCleared("森林"))
	})

	t.Run("イベント状態の保存と復元", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})

		// イベント状態を設定
		worldhelper.GetGameProgress(world).SetEventActive("all_cleared")
		worldhelper.GetGameProgress(world).MarkEventSeen("all_cleared")

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		err = sm.RestoreWorldFromJSON(newWorld, jsonStr)
		require.NoError(t, err)

		// 視聴済みイベントはIsEventUnseenがfalseになる
		assert.False(t, worldhelper.GetGameProgress(newWorld).IsEventUnseen("all_cleared"))
		ev := worldhelper.GetGameProgress(newWorld).Events["all_cleared"]
		assert.True(t, ev.Active)
		assert.True(t, ev.Seen)
	})

	t.Run("空のGameProgressの保存と復元", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		err = sm.RestoreWorldFromJSON(newWorld, jsonStr)
		require.NoError(t, err)

		assert.Empty(t, worldhelper.GetGameProgress(newWorld).ClearedDungeons)
		assert.Empty(t, worldhelper.GetGameProgress(newWorld).Events)
	})
}
