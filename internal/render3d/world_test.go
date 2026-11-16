package render3d_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorldProjector_ワールドのカメラとプレイヤー位置から投影を組む(t *testing.T) {
	t.Parallel()

	// プレイヤーの spawn がカメラを既定の3Dオービットごと用意する
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(screenW, screenH)
	_, err := lifecycle.SpawnPlayer(world, playerTile, "ash")
	require.NoError(t, err)

	// 同じ状態からは同じ投影が組める。呼び出し側ごとに寸法の取り方が分かれないことを固定する
	projector, err := render3d.WorldProjector(world)
	require.NoError(t, err)
	assert.Equal(t, render3d.NewProjector(defaultView, playerTile, screenW, screenH), projector)
}

func TestWorldProjector_カメラを回すと投影が追随する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(screenW, screenH)
	_, err := lifecycle.SpawnPlayer(world, playerTile, "ash")
	require.NoError(t, err)
	camera := query.GetPlayerCamera(world)
	require.NotNil(t, camera)
	far := consts.Coord[consts.Tile]{X: 25, Y: 20}

	projector, err := render3d.WorldProjector(world)
	require.NoError(t, err)
	north, ok := projector.TileCenter(far, 0)
	require.True(t, ok)
	camera.Orient = 1
	projector, err = render3d.WorldProjector(world)
	require.NoError(t, err)
	rotated, ok := projector.TileCenter(far, 0)
	require.True(t, ok)

	assert.Greater(t, float64(rotated.X), float64(north.X)+100)
}

func TestWorldProjector_カメラがいないとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(screenW, screenH)

	_, err := render3d.WorldProjector(world)
	require.ErrorIs(t, err, render3d.ErrNoPlayerCamera)
}

func TestWorldProjector_プレイヤーが複数いるとエラー(t *testing.T) {
	t.Parallel()

	// プレイヤーが2人いると投影の注視点を一意に決められずエラーになる
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(screenW, screenH)
	_, err := lifecycle.SpawnPlayer(world, playerTile, "ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 26, Y: 25}, "ash")
	require.NoError(t, err)

	_, err = render3d.WorldProjector(world)
	require.ErrorIs(t, err, query.ErrMultiplePlayers)
}

func TestPlayerTile_プレイヤーがいないとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	_, err := render3d.PlayerTile(world)
	require.ErrorIs(t, err, query.ErrNoPlayer)
}

func TestTileTopHeight_壁は天面床は地面の高さになる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnTile(world, "wall", 25, 24, nil)
	require.NoError(t, err)
	_, err = lifecycle.SpawnTile(world, "floor", 25, 23, nil)
	require.NoError(t, err)

	// 壁は箱として描かれるので、カーソルは天面へ描かないと箱に埋もれる
	assert.InDelta(t, render3d.WallHeight, render3d.TileTopHeight(world, consts.Coord[consts.Tile]{X: 25, Y: 24}), 1e-9)
	assert.InDelta(t, 0.0, render3d.TileTopHeight(world, consts.Coord[consts.Tile]{X: 25, Y: 23}), 1e-9)
	assert.InDelta(t, 0.0, render3d.TileTopHeight(world, consts.Coord[consts.Tile]{X: 40, Y: 40}), 1e-9)
}

func TestIsBox_通行を塞ぐだけの物は箱にならない(t *testing.T) {
	t.Parallel()

	// 樽は BlockPass を持つが、壁にはめる形でないので箱として描かれない。
	// 通行不能をそのまま壁とみなすと、箱の無い場所へカーソルが浮く
	world := testutil.InitTestWorld(t)
	barrel, err := lifecycle.SpawnProp(world, "barrel", 25, 24)
	require.NoError(t, err)
	require.True(t, world.Components.BlockPass.Has(barrel))

	assert.False(t, render3d.IsBox(world, consts.Coord[consts.Tile]{X: 25, Y: 24}))
}

func TestIsBox_壁にはめる扉は閉じている間だけ箱になる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	pos := consts.Coord[consts.Tile]{X: 25, Y: 24}
	door, err := lifecycle.SpawnDoor(world, pos, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	assert.True(t, render3d.IsBox(world, pos))

	require.NoError(t, lifecycle.OpenDoor(world, door))
	assert.False(t, render3d.IsBox(world, pos))
}

func TestBoxSet_箱になるものと箱になるタイルを集める(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnTile(world, "wall", 25, 24, nil)
	require.NoError(t, err)
	_, err = lifecycle.SpawnTile(world, "floor", 25, 23, nil)
	require.NoError(t, err)
	_, err = lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 26, Y: 24}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)

	// 箱の側面を省く判定と、タイルを指す判定が同じ集合を見る
	boxes, tiles := render3d.BoxSet(world)
	assert.Equal(t, map[consts.Coord[consts.Tile]]bool{{X: 25, Y: 24}: true, {X: 26, Y: 24}: true}, boxes)
	assert.Equal(t, map[consts.Coord[consts.Tile]]bool{{X: 25, Y: 24}: true}, tiles)
}
