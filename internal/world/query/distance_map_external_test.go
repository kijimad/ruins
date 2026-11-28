package query_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HP を持つ置物の無い盤面で、地図の斜めの角の規則が activity.CanMoveTo と食い違わないことを確かめる。HP を持つ置物の
// マスは、地図では壊しに行く下る先になり、CanMoveTo では通れないので、この盤面には置かない
func TestGetCubeDistanceMap_斜めの隣はCanMoveToでも通れる(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	world := testutil.InitTestWorld(t)
	for _, c := range []consts.Coord[consts.Tile]{at(0, 0), at(7, 7)} {
		_, err := lifecycle.SpawnCube(world, c)
		require.NoError(t, err)
	}
	si := query.GetSpatialIndex(world)
	si.Built, si.MapWidth, si.MapHeight = true, 8, 8
	si.Characters = map[gc.GridElement]ecs.Entity{}
	si.Blocked = map[gc.GridElement]bool{}
	for _, c := range []consts.Coord[consts.Tile]{at(2, 1), at(1, 2), at(4, 3), at(3, 4), at(5, 5), at(6, 2), at(2, 6)} {
		si.Blocked[gc.GridElement{Coord: c}] = true
	}

	m := query.GetCubeDistanceMap(world)
	checked := 0
	for y := range 8 {
		for x := range 8 {
			from := at(x, y)
			for _, next := range m.Steps(from, nil) {
				if next.X != from.X && next.Y != from.Y {
					checked++
					assert.True(t, activity.CanMoveTo(world, next, from, ecs.Entity{}), "%v から %v へ", from, next)
				}
			}
		}
	}
	require.Positive(t, checked, "前提: 斜めの隣を確かめている")
}
