package save

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSerdeWholeWorldRoundtrip は代表的なエンティティ（プレイヤー・敵・アイテム・扉・置物）を
// 配置し、production の SaveWorld/LoadWorld 経由で丸ごと保存・復元されることを検証する。
func TestSerdeWholeWorldRoundtrip(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	_, err = lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnEnemy(world, 8, 8, "火の玉")
	require.NoError(t, err)
	// 回復薬は平坦化された ProvidesHealing を持つ。serde 往復で保存されることを確認する
	_, err = lifecycle.SpawnFieldItem(world, "回復薬", 3, 3, 2)
	require.NoError(t, err)
	_, err = lifecycle.SpawnDoor(world, 4, 4, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "木箱", 6, 6)
	require.NoError(t, err)

	require.NoError(t, manager.SaveWorld(world, "roundtrip"))

	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "roundtrip"))

	// プレイヤーが復元される
	playerCount := 0
	pq := ecs.NewFilter1[gc.Player](newWorld.World).Query()
	for pq.Next() {
		playerCount++
	}
	assert.Equal(t, 1, playerCount, "プレイヤーが1体復元される")

	// 敵（丸ごと保存のため復元される）
	enemyCount := 0
	eq := ecs.NewFilter1[gc.SoloAI](newWorld.World).Query()
	for eq.Next() {
		enemyCount++
	}
	assert.GreaterOrEqual(t, enemyCount, 1, "敵が復元される")

	// 扉（地形も丸ごと保存される）
	doorCount := 0
	dq := ecs.NewFilter1[gc.Door](newWorld.World).Query()
	for dq.Next() {
		doorCount++
	}
	assert.Equal(t, 1, doorCount, "扉が復元される")

	// ProvidesHealing が復元される（平坦化により serde 可能になった）
	healingFound := false
	hq := ecs.NewFilter1[gc.ProvidesHealing](newWorld.World).Query()
	for hq.Next() {
		healingFound = true
	}
	assert.True(t, healingFound, "回復効果コンポーネントが復元される")
}

// TestSerde_DungeonLocationPersists は現在地（Dungeonの階層・定義名）が保存・復元されることを検証する。
// ロード時の復帰先ステート再構築はこの2フィールドに依存する。
func TestSerde_DungeonLocationPersists(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	_, err = lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	dungeonState := query.GetDungeon(world)
	dungeonState.Depth = 3
	dungeonState.DefinitionName = "遺跡"

	require.NoError(t, manager.SaveWorld(world, "location"))

	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "location"))

	restored := query.GetDungeon(newWorld)
	require.NotNil(t, restored)
	assert.Equal(t, 3, restored.Depth, "階層が復元される")
	assert.Equal(t, "遺跡", restored.DefinitionName, "ダンジョン定義名が復元される")
}
