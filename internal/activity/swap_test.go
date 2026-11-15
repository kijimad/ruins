package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteMoveAction_キューブへ歩き込むと位置を入れ替える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 6, Y: 5})
	require.NoError(t, err)

	require.NoError(t, ExecuteMoveAction(world, gc.DirectionRight))

	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(player).Coord)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord)
}

func TestExecuteMoveAction_展開中のキューブとは入れ替えない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 6, Y: 5})
	require.NoError(t, err)
	require.True(t, lifecycle.DeployCube(world, cube))

	require.NoError(t, ExecuteMoveAction(world, gc.DirectionRight))

	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(player).Coord)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord)
	entries := query.GetGameLog(world).GetRecentEntries(1)
	require.Len(t, entries, 1)
	assert.Equal(t, query.T(world, "The deployed cube cannot be moved."), entries[0].Text())
}
