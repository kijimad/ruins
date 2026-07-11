package save

import (
	"os"
	"path/filepath"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
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
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})

	npc := world.ECS.NewEntity()
	world.Components.Name.Add(npc, &gc.Name{Name: "テストNPC"})
	world.Components.FactionEnemy.Add(npc, &gc.FactionEnemyData{})

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

	npcQuery := ecs.NewFilter1[gc.FactionEnemyData](newWorld.ECS).Query()
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
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})

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
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})

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

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})

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

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})

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

// TestSaveLoadSquadMember は隊員のセーブ/ロード往復で必要なコンポーネントが復元されることを検証する
func TestSaveLoadSquadMember(t *testing.T) {
	t.Parallel()

	abilities := gc.Abilities{
		Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
		Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
	}

	t.Run("隊員のランタイムコンポーネントが復元される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnSquadMember(world, player, "隊員A", abilities, "player")
		require.NoError(t, err)

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		err = sm.RestoreWorldFromJSON(newWorld, jsonStr)
		require.NoError(t, err)

		// 復元後の隊員を検索する
		var memberEntity ecs.Entity
		var found bool
		memberQuery := ecs.NewFilter1[gc.SquadMember](newWorld.ECS).Query()
		for memberQuery.Next() {
			memberEntity = memberQuery.Entity()
			found = true
		}
		require.True(t, found, "隊員エンティティが復元されている")

		// AI処理に必要なコンポーネント
		assert.True(t, newWorld.Components.SquadAI.Has(memberEntity), "SquadAIが復元される")
		assert.True(t, newWorld.Components.GridElement.Has(memberEntity), "GridElementが復元される")

		// ステータス関連コンポーネント
		assert.True(t, newWorld.Components.HealthStatus.Has(memberEntity), "HealthStatusが復元される")
		assert.True(t, newWorld.Components.Skills.Has(memberEntity), "Skillsが復元される")

		// AIの値が正しいことを確認
		ai := newWorld.Components.SquadAI.Get(memberEntity)
		assert.Equal(t, gc.PlannerSquad, ai.Type())
		assert.Equal(t, gc.CombatAttack, ai.CombatCurrent)
	})

	// SquadMember は空マーカーでリーダー参照フィールドを持たない（leader は配置位置の
	// 決定にのみ使う）。ここではプレイヤーと隊員の両エンティティが独立に復元されることを検証する。
	// エンティティ参照の再マッピング自体は unified_test の assertComplexWorldRestored で検証する
	t.Run("プレイヤーと隊員の両エンティティが復元される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnSquadMember(world, player, "隊員A", abilities, "player")
		require.NoError(t, err)

		sm := createTestSerializationManager(t)
		jsonStr, err := sm.GenerateWorldJSON(world)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		err = sm.RestoreWorldFromJSON(newWorld, jsonStr)
		require.NoError(t, err)

		// プレイヤーが復元されていること
		var playerFound bool
		playerQuery := ecs.NewFilter1[gc.Player](newWorld.ECS).Query()
		for playerQuery.Next() {
			playerFound = true
		}
		assert.True(t, playerFound, "プレイヤーが復元されている")

		// 隊員マーカーが復元されていること
		var memberFound bool
		memberQuery := ecs.NewFilter1[gc.SquadMember](newWorld.ECS).Query()
		for memberQuery.Next() {
			memberFound = true
		}
		assert.True(t, memberFound, "隊員マーカーが復元されている")
	})
}
