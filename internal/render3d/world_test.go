package render3d_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newWorld はタイル(25,25)にプレイヤーが立つワールドを返す。
// プレイヤーの spawn がカメラを既定の3Dオービットごと用意する。
func newWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(screenW, screenH)
	_, err := lifecycle.SpawnPlayer(world, playerTile, "ash")
	require.NoError(t, err)
	return world
}

func TestFor_ワールドのカメラとプレイヤー位置から投影を組む(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	// 同じ状態からは同じ投影が組める。呼び出し側ごとに寸法の取り方が分かれないことを固定する
	assert.Equal(t, render3d.NewProjector(defaultView, playerTile, screenW, screenH), render3d.ProjectorFor(world))
}

func TestFor_カメラが無くても既定の視点で組む(t *testing.T) {
	t.Parallel()

	// 投影を諦めるとカーソルが消え、位置がずれるより分かりにくい壊れ方になる
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(screenW, screenH)
	require.Nil(t, query.GetPlayerCamera(world))

	_, ok := render3d.ProjectorFor(world).TileCenter(playerTile, 0)
	assert.True(t, ok)
}

func TestFor_カメラを回すと投影が追随する(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	camera := query.GetPlayerCamera(world)
	require.NotNil(t, camera)
	far := consts.Coord[consts.Tile]{X: 25, Y: 20}

	north, ok := render3d.ProjectorFor(world).TileCenter(far, 0)
	require.True(t, ok)
	camera.Orient = 1
	rotated, ok := render3d.ProjectorFor(world).TileCenter(far, 0)
	require.True(t, ok)

	assert.Greater(t, float64(rotated.X), float64(north.X)+100)
}

func TestTileTopHeight_壁は天面床は地面の高さになる(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	_, err := lifecycle.SpawnTile(world, "wall", 25, 24, nil)
	require.NoError(t, err)
	_, err = lifecycle.SpawnTile(world, "floor", 25, 23, nil)
	require.NoError(t, err)

	// 壁は箱として描かれるので、カーソルは天面へ描かないと箱に埋もれる
	assert.InDelta(t, render3d.WallHeight, render3d.TileTopHeight(world, consts.Coord[consts.Tile]{X: 25, Y: 24}), 1e-9)
	assert.InDelta(t, 0.0, render3d.TileTopHeight(world, consts.Coord[consts.Tile]{X: 25, Y: 23}), 1e-9)
	assert.InDelta(t, 0.0, render3d.TileTopHeight(world, consts.Coord[consts.Tile]{X: 40, Y: 40}), 1e-9)
}

func TestIsWallTile_通行を塞ぐだけの物は箱にならない(t *testing.T) {
	t.Parallel()

	// 扉は BlockPass を持つが Tile ではないので箱として描かれない。
	// 通行不能をそのまま壁とみなすと、箱の無い場所へカーソルが浮く
	world := newWorld(t)
	door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 25, Y: 24}, 0)
	require.NoError(t, err)
	require.True(t, world.Components.BlockPass.Has(door))

	assert.False(t, render3d.IsWallTile(world, consts.Coord[consts.Tile]{X: 25, Y: 24}))
}

func TestWallTileSet_箱になるタイルだけを集める(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	_, err := lifecycle.SpawnTile(world, "wall", 25, 24, nil)
	require.NoError(t, err)
	_, err = lifecycle.SpawnTile(world, "floor", 25, 23, nil)
	require.NoError(t, err)

	// 壁の側面を省く判定と、タイルを指す判定が同じ集合を見る
	walls := render3d.WallTileSet(world)
	assert.True(t, walls[consts.Coord[consts.Tile]{X: 25, Y: 24}])
	assert.False(t, walls[consts.Coord[consts.Tile]{X: 25, Y: 23}])
}
