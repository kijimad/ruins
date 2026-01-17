package mapspawner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestTownPlannerIntegration(t *testing.T) {
	t.Parallel()

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)
	world.Resources.RawMaster = createTestRawMaster(t)

	// 街マップ生成（Plan）
	width, height := 50, 50
	seed := uint64(42)
	plannerType := mapplanner.PlannerTypeTown

	metaPlan, err := mapplanner.Plan(world, width, height, seed, plannerType)
	require.NoError(t, err, "Plan failed")
	require.NotNil(t, metaPlan, "MetaPlan should not be nil")

	// MetaPlanからエンティティ生成（Spawn）
	level, err := Spawn(world, metaPlan)
	require.NoError(t, err, "SpawnFromMetaPlan failed")

	// レベルの基本プロパティを確認
	require.Equal(t, gc.Tile(50), level.TileWidth, "TileWidth should be 50")
	require.Equal(t, gc.Tile(50), level.TileHeight, "TileHeight should be 50")

	// プレイヤー開始位置を確認
	playerX, playerY, hasPlayer := metaPlan.GetPlayerStartPosition()
	require.True(t, hasPlayer, "Should have player start position")
	require.GreaterOrEqual(t, playerX, 0, "Player X should be valid")
	require.GreaterOrEqual(t, playerY, 0, "Player Y should be valid")
	require.Less(t, playerX, 50, "Player X should be within bounds")
	require.Less(t, playerY, 50, "Player Y should be within bounds")

	// Propsとワープポータルが配置されているか確認
	require.Greater(t, len(metaPlan.Props), 0, "Should have Props")
	require.Greater(t, len(metaPlan.WarpPortals), 0, "Should have at least one warp portal")
	require.Greater(t, len(metaPlan.Doors), 0, "Should have Doors")
	// NPCは現在スキップ中

	// GridElementコンポーネントを持つエンティティ数を確認
	entityCount := world.Manager.Join(world.Components.GridElement).Size()
	require.Greater(t, entityCount, 1000, "Should have many entities spawned (walls, floors, etc)")

	// ドアエンティティの数を確認
	doorCount := world.Manager.Join(world.Components.Door).Size()
	require.Equal(t, len(metaPlan.Doors), doorCount, "Spawned door count should match planned door count")

	t.Logf("街マップ統合テスト成功: エンティティ数=%d, ドア数=%d, プレイヤー位置=(%d,%d)",
		entityCount, doorCount, playerX, playerY)
}

func TestTownPlannerVsSmallRoom(t *testing.T) {
	t.Parallel()

	// 同じワールドでSmallRoomとTownを比較
	world := testutil.InitTestWorld(t)
	world.Resources.RawMaster = createTestRawMaster(t)

	seed := uint64(123)

	// SmallRoomプランナー
	smallRoomPlan, err := mapplanner.Plan(world, 20, 20, seed, mapplanner.PlannerTypeSmallRoom)
	require.NoError(t, err, "SmallRoom Plan failed")

	// Townプランナー
	townPlan, err := mapplanner.Plan(world, 50, 50, seed, mapplanner.PlannerTypeTown)
	require.NoError(t, err, "Town Plan failed")

	// サイズの違いを確認
	require.Equal(t, gc.Tile(20), smallRoomPlan.Level.TileWidth, "SmallRoom should be 20x20")
	require.Equal(t, gc.Tile(50), townPlan.Level.TileWidth, "Town should be 50x50")

	// NPCの配置状況を確認
	// Townマップには4体配置される
	require.Equal(t, 4, len(townPlan.NPCs), "Town should have 4 conversation NPCs")
	// SmallRoomのNPC数は制限しない（ランダム生成のため）

	// Propsの配置の違いを確認
	require.Greater(t, len(townPlan.Props), len(smallRoomPlan.Props), "Town should have more props than SmallRoom")

	t.Logf("SmallRoom: NPC=%d, Props=%d", len(smallRoomPlan.NPCs), len(smallRoomPlan.Props))
	t.Logf("Town: NPC=%d, Props=%d", len(townPlan.NPCs), len(townPlan.Props))
}
