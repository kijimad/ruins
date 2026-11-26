package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpatialIndex_地形と置物が変わるたびに全走査と同じ答えを返す は、扉の開閉・置物の生成と破壊・設備を置く・戻すの
// それぞれの後で、ターンの終わりを待たずに、通れないマスと移動コストの表が全走査の答えと一致することを確かめる
func TestSpatialIndex_地形と置物が変わるたびに全走査と同じ答えを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, lifecycle.DeployCube(world, cube))

	matchesFullScan := func(step string) {
		t.Helper()
		wantBlocked := map[gc.GridElement]bool{}
		query.ForEachTerrainOrPropWithout(world, func(e ecs.Entity) {
			wantBlocked[*world.Components.GridElement.Get(e)] = true
		}, ecs.C[gc.Passable]())
		wantCosts := map[gc.GridElement]int{}
		query.ForEachTerrainOrPropWithout(world, func(e ecs.Entity) {
			if !world.Components.Passable.Has(e) {
				return
			}
			if cost := world.Components.Passable.Get(e).Cost; cost > 0 {
				wantCosts[*world.Components.GridElement.Get(e)] += cost
			}
		})

		si := query.GetSpatialIndex(world)
		require.NotNil(t, si, step)
		query.PassCostAt(world, consts.Coord[consts.Tile]{})
		assert.Equal(t, wantBlocked, si.Blocked, step)
		assert.Equal(t, wantCosts, si.PassCosts, step)
	}
	matchesFullScan("最初")

	stone, err := lifecycle.SpawnProp(world, "moving_stone", 7, 5)
	require.NoError(t, err)
	matchesFullScan("置物の生成")

	door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 8, Y: 5}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	matchesFullScan("扉の生成")
	require.NoError(t, lifecycle.OpenDoor(world, door))
	matchesFullScan("扉を開く")
	require.NoError(t, lifecycle.CloseDoor(world, door))
	matchesFullScan("扉を閉じる")

	gameaction.ApplyDamage(world, stone, world.Components.HP.Get(stone).Current, player)
	require.True(t, world.Components.Dead.Has(stone))
	matchesFullScan("置物の破壊")

	item, err := lifecycle.SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	_, err = lifecycle.PlaceFacility(world, cube, item, consts.Coord[consts.Tile]{X: 11, Y: 10})
	require.NoError(t, err)
	matchesFullScan("設備を置く")
	require.NoError(t, lifecycle.RemoveFacility(world, item, player))
	matchesFullScan("設備を戻す")
}
