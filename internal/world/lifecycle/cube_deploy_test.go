package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployCube(t *testing.T) {
	t.Parallel()

	t.Run("周囲が空いていれば展開する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
		require.NoError(t, err)

		assert.True(t, DeployCube(world, cube))
		assert.True(t, world.Components.Deployed.Has(cube), "展開中マーカーが付く")
	})

	t.Run("塞がれていれば展開しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
		require.NoError(t, err)
		// 隣を通行不可にすると全か無かで展開が拒否される
		blocker := world.ECS.NewEntity()
		world.Components.GridElement.Add(blocker, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 9}})
		world.Components.BlockPass.Add(blocker, &gc.BlockPass{})
		query.InvalidateSpatialIndex(world)

		assert.False(t, DeployCube(world, cube))
		assert.False(t, world.Components.Deployed.Has(cube), "展開しないのでマーカーは付かない")
	})

	t.Run("既に展開中なら真を返し変えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
		require.NoError(t, err)
		world.Components.Deployed.Add(cube, &gc.Deployed{})

		assert.True(t, DeployCube(world, cube))
	})
}

func TestStowCube(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})

	StowCube(world, cube)
	assert.False(t, world.Components.Deployed.Has(cube), "収納で展開中マーカーが外れる")
}
