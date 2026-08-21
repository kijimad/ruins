package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectionScreen は投影テストで使う画面寸法。ゴールデンと同じ大きさにそろえる
const (
	projectionScreenW = 960
	projectionScreenH = 720
)

// newProjectionWorld はタイル(25,25)にプレイヤーが立つワールドを返す。
// プレイヤーの spawn がカメラを既定の3Dオービットごと用意する。
func newProjectionWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(projectionScreenW, projectionScreenH)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 25, Y: 25}, "ash")
	require.NoError(t, err)
	return world
}

func TestProjector_足元のタイル中心は画面中央の真下に写る(t *testing.T) {
	t.Parallel()

	world := newProjectionWorld(t)
	projector := NewProjector(world, projectionScreenW, projectionScreenH)

	center, ok := projector.TileCenter(consts.Coord[consts.Tile]{X: 25, Y: 25}, 0)
	require.True(t, ok)
	// カメラは高さ 0.4 を見つめるので、地面は画面中央より下に写る
	assert.InDelta(t, 480.0, float64(center.X), 0.05)
	assert.InDelta(t, 374.8, float64(center.Y), 0.05)
}

func TestProjector_カメラを回すとタイルの写る位置が変わる(t *testing.T) {
	t.Parallel()

	world := newProjectionWorld(t)
	camera := query.GetPlayerCamera(world)
	require.NotNil(t, camera)
	tile := consts.Coord[consts.Tile]{X: 25, Y: 20}

	north, ok := NewProjector(world, projectionScreenW, projectionScreenH).TileCenter(tile, 0)
	require.True(t, ok)

	camera.Orient = 1
	rotated, ok := NewProjector(world, projectionScreenW, projectionScreenH).TileCenter(tile, 0)
	require.True(t, ok)

	// 真北を向いていれば奥のタイルは画面中央の真上、45度回すと右へ寄る
	assert.InDelta(t, 480.0, float64(north.X), 0.05)
	assert.Greater(t, float64(rotated.X), float64(north.X)+100)
}

func TestProjector_四隅は北西から時計回りに並ぶ(t *testing.T) {
	t.Parallel()

	world := newProjectionWorld(t)
	projector := NewProjector(world, projectionScreenW, projectionScreenH)

	corners, ok := projector.TileCorners(consts.Coord[consts.Tile]{X: 25, Y: 25}, 0)
	require.True(t, ok)

	nw, ne, se, sw := corners[0], corners[1], corners[2], corners[3]
	assert.Less(t, float64(nw.X), float64(ne.X), "北西は北東より左")
	assert.Less(t, float64(sw.X), float64(se.X), "南西は南東より左")
	assert.Less(t, float64(nw.Y), float64(sw.Y), "北辺は南辺より上")
	// 手前ほど大きく写るので、南辺は北辺より横に広い
	assert.Greater(t, float64(se.X-sw.X), float64(ne.X-nw.X))
}

func TestProjector_高さを上げると画面の上へ写る(t *testing.T) {
	t.Parallel()

	world := newProjectionWorld(t)
	projector := NewProjector(world, projectionScreenW, projectionScreenH)
	tile := consts.Coord[consts.Tile]{X: 25, Y: 24}

	ground, ok := projector.TileCenter(tile, 0)
	require.True(t, ok)
	top, ok := projector.TileCenter(tile, WallHeight)
	require.True(t, ok)

	assert.Less(t, float64(top.Y), float64(ground.Y), "壁の天面は地面より上に写る")
}

func TestProjector_カメラ後方のタイルは投影できない(t *testing.T) {
	t.Parallel()

	world := newProjectionWorld(t)
	projector := NewProjector(world, projectionScreenW, projectionScreenH)

	// カメラはプレイヤーの南側にいる。そのさらに南は視線の裏側に回る
	_, ok := projector.TileCenter(consts.Coord[consts.Tile]{X: 25, Y: 60}, 0)
	assert.False(t, ok)
}

func TestNewProjector_カメラが無くても既定の視点で組む(t *testing.T) {
	t.Parallel()

	// カメラを持つエンティティがいないワールド。投影を諦めるとカーソルが消え、
	// 位置がずれるより分かりにくい壊れ方になるので、既定の視点で組み続ける
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(projectionScreenW, projectionScreenH)
	require.Nil(t, query.GetPlayerCamera(world))

	_, ok := NewProjector(world, projectionScreenW, projectionScreenH).
		TileCenter(consts.Coord[consts.Tile]{X: 25, Y: 25}, 0)
	assert.True(t, ok)
}

func TestNewProjector_3D値が未設定のカメラでも既定の視点で組む(t *testing.T) {
	t.Parallel()

	// 3Dの値を持たないセーブから復元するとゼロ値になる。そのまま使うと真横から見る絵になるので、
	// 投影を組む時点でも既定へ寄せる
	world := newProjectionWorld(t)
	camera := query.GetPlayerCamera(world)
	require.NotNil(t, camera)
	camera.Pitch, camera.Dist = 0, 0

	center, ok := NewProjector(world, projectionScreenW, projectionScreenH).
		TileCenter(consts.Coord[consts.Tile]{X: 25, Y: 25}, 0)
	require.True(t, ok)
	assert.InDelta(t, 374.8, float64(center.Y), 0.05)
}

func TestProjector_世界描画と同じ投影行列を使う(t *testing.T) {
	t.Parallel()

	// 世界レイヤの床クアッドと、その上へ重ねるカーソルが同じ場所に来ることを固定する。
	// 行列の組み立てが2箇所に分かれると、片方だけ直して片方が取り残される
	world := newProjectionWorld(t)
	camera := query.GetPlayerCamera(world)
	require.NotNil(t, camera)
	camera.Orient = 3
	camera.Pitch = 0.9

	sys := &Render3DSystem{UseFOV: false}
	_, vp, _ := sys.buildScene(world, projectionScreenW, projectionScreenH)
	projector := NewProjector(world, projectionScreenW, projectionScreenH)

	require.Equal(t, vp, projector.vp, "世界描画とオーバーレイが同じ view-projection 行列を使う")

	// 行列が同じなら、床クアッドの四隅とカーソル枠の四隅も一致する
	tile := consts.Coord[consts.Tile]{X: 25, Y: 24}
	corners, ok := projector.TileCorners(tile, 0)
	require.True(t, ok)
	quad := [4]r3vec{{25, 0, 24}, {26, 0, 24}, {26, 0, 25}, {25, 0, 25}}
	for i, p := range quad {
		x, y, projected := projectToScreen(vp, p, projectionScreenW, projectionScreenH)
		require.True(t, projected)
		assert.InDelta(t, x, float64(corners[i].X), 1e-9)
		assert.InDelta(t, y, float64(corners[i].Y), 1e-9)
	}
}

func TestTileTopHeight_壁は天面床は地面の高さになる(t *testing.T) {
	t.Parallel()

	world := newProjectionWorld(t)
	_, err := lifecycle.SpawnTile(world, "wall", 25, 24, nil)
	require.NoError(t, err)
	_, err = lifecycle.SpawnTile(world, "floor", 25, 23, nil)
	require.NoError(t, err)

	// 壁は箱として描かれるので、カーソルは天面へ描かないと箱に埋もれる
	assert.InDelta(t, WallHeight, TileTopHeight(world, consts.Coord[consts.Tile]{X: 25, Y: 24}), 1e-9)
	assert.InDelta(t, 0.0, TileTopHeight(world, consts.Coord[consts.Tile]{X: 25, Y: 23}), 1e-9)
	assert.InDelta(t, 0.0, TileTopHeight(world, consts.Coord[consts.Tile]{X: 40, Y: 40}), 1e-9)
}

func TestProjector_BillboardScaleは立て板の画面上の高さを返す(t *testing.T) {
	t.Parallel()

	world := newProjectionWorld(t)
	projector := NewProjector(world, projectionScreenW, projectionScreenH)
	near := consts.Coord[consts.Tile]{X: 25, Y: 25}
	far := consts.Coord[consts.Tile]{X: 25, Y: 20}

	nearScale, ok := projector.BillboardScale(near)
	require.True(t, ok)
	farScale, ok := projector.BillboardScale(far)
	require.True(t, ok)

	assert.Greater(t, nearScale, 0.0)
	assert.Greater(t, nearScale, farScale, "手前の立て板ほど大きく写る")
}

func TestCameraYaw_Orientから水平角を導出する(t *testing.T) {
	t.Parallel()

	// 水平角を別フィールドに持たせず Orient から導くので、2箇所に分かれてずれることがない
	assert.InDelta(t, 0.0, gc.Camera{Orient: 0}.Yaw(), 1e-9)
	assert.InDelta(t, 0.7853981633974483, gc.Camera{Orient: 1}.Yaw(), 1e-9)
	assert.InDelta(t, 5.497787143782138, gc.Camera{Orient: 7}.Yaw(), 1e-9)
}
