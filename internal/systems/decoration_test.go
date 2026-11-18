package systems

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/require"
)

// TestItemMarkerTiles はマーカーを出すマスの判定を、視界・投影・描画から切り離して固定する。
func TestItemMarkerTiles(t *testing.T) {
	t.Parallel()

	all := func(consts.Coord[consts.Tile]) bool { return true }

	t.Run("異なる品種が重なったマスには出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		c := consts.Coord[consts.Tile]{X: 3, Y: 3}
		_, err := lifecycle.SpawnFieldItem(world, "healing_potion", c.X, c.Y, 1)
		require.NoError(t, err)
		_, err = lifecycle.SpawnFieldItem(world, "bread", c.X, c.Y, 1)
		require.NoError(t, err)

		markers := itemMarkerTiles(world, all)
		require.True(t, markers[c])
	})

	t.Run("同種スタックのマスには出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		c := consts.Coord[consts.Tile]{X: 3, Y: 3}
		_, err := lifecycle.SpawnFieldItem(world, "healing_potion", c.X, c.Y, 3)
		require.NoError(t, err)

		markers := itemMarkerTiles(world, all)
		require.False(t, markers[c])
	})

	t.Run("単品のマスには出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		c := consts.Coord[consts.Tile]{X: 3, Y: 3}
		_, err := lifecycle.SpawnFieldItem(world, "healing_potion", c.X, c.Y, 1)
		require.NoError(t, err)

		markers := itemMarkerTiles(world, all)
		require.False(t, markers[c])
	})

	t.Run("まだ探していない収納のマスには、空でも出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		c := consts.Coord[consts.Tile]{X: 4, Y: 4}
		closet, err := lifecycle.SpawnProp(world, "closet", c.X, c.Y)
		require.NoError(t, err)
		require.True(t, world.Components.Unsearched.Has(closet))

		markers := itemMarkerTiles(world, all)
		require.True(t, markers[c])
	})

	t.Run("探し終えた中身のある収納のマスには出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		c := consts.Coord[consts.Tile]{X: 4, Y: 4}
		closet, err := lifecycle.SpawnProp(world, "closet", c.X, c.Y)
		require.NoError(t, err)
		_, err = lifecycle.SpawnStorageItem(world, "healing_potion", 1, closet)
		require.NoError(t, err)
		world.Components.Unsearched.Remove(closet)

		markers := itemMarkerTiles(world, all)
		require.True(t, markers[c])
	})

	t.Run("探し終えた空の収納のマスには出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		c := consts.Coord[consts.Tile]{X: 4, Y: 4}
		closet, err := lifecycle.SpawnProp(world, "closet", c.X, c.Y)
		require.NoError(t, err)
		world.Components.Unsearched.Remove(closet)

		markers := itemMarkerTiles(world, all)
		require.False(t, markers[c])
	})

	t.Run("within の外のマスは調べない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		inside := consts.Coord[consts.Tile]{X: 1, Y: 1}
		outside := consts.Coord[consts.Tile]{X: 9, Y: 9}
		// outside に本来出る条件を置いても、within が false のマスは拾わない
		_, err := lifecycle.SpawnFieldItem(world, "healing_potion", outside.X, outside.Y, 1)
		require.NoError(t, err)
		_, err = lifecycle.SpawnFieldItem(world, "bread", outside.X, outside.Y, 1)
		require.NoError(t, err)

		only := func(c consts.Coord[consts.Tile]) bool { return c == inside }
		markers := itemMarkerTiles(world, only)
		require.False(t, markers[outside])
	})
}
