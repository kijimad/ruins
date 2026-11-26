package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpatialIndex_地形と置物が変わるたびに全走査と同じ答えを返す は、扉の開閉・置物の生成と破壊・設備を置く・戻すの
// キューブの畳み込みと展開の、それぞれの後で、ターンの終わりを待たずに、通れないマスと移動コストの表が全走査の答えと一致することを確かめる
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
		// 索引と別の道で求めるため、座標を持つ実体を全て回し、壊れていない通り方を持つ実体を1件ずつ見る
		wantBlocked := map[gc.GridElement]bool{}
		wantCosts := map[gc.GridElement]int{}
		q := query.ActiveFilter1[gc.GridElement](world).Query()
		for q.Next() {
			e := q.Entity()
			if !world.Components.Passage.Has(e) || world.Components.Dead.Has(e) {
				continue
			}
			ge := *world.Components.GridElement.Get(e)
			passage := world.Components.Passage.Get(e)
			if !passage.Passable {
				wantBlocked[ge] = true
			} else if passage.Cost > 0 {
				wantCosts[ge] += passage.Cost
			}
		}

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

	// 野営の中の置物は畳むと床から外れ、展開すると戻る
	_, err = lifecycle.SpawnProp(world, "fence", 9, 10)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "moving_stone", 10, 11)
	require.NoError(t, err)
	matchesFullScan("野営に置物を置く")
	lifecycle.StowCube(world, cube)
	matchesFullScan("キューブを畳む")
	require.True(t, lifecycle.DeployCube(world, cube))
	matchesFullScan("キューブを展開する")
	require.NoError(t, lifecycle.RemoveFacility(world, item, player))
	matchesFullScan("設備を戻す")
}
