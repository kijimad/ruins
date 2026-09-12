package render3d_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 投影テストの条件。ゴールデンと同じ画面寸法とタイル位置にそろえる
const (
	screenW = 960
	screenH = 720
)

var (
	playerTile  = consts.Coord[consts.Tile]{X: 25, Y: 25}
	defaultView = gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}
)

func TestNew_足元のタイル中心は画面中央の真下に写る(t *testing.T) {
	t.Parallel()

	center, ok := render3d.NewProjector(defaultView, playerTile, screenW, screenH).TileCenter(playerTile, 0)
	require.True(t, ok)
	// カメラは高さ 0.4 を見つめるので、地面は画面中央より下に写る
	assert.InDelta(t, 480.0, float64(center.X), 0.05)
	assert.InDelta(t, 374.8, float64(center.Y), 0.05)
}

func TestNew_カメラを回すとタイルの写る位置が変わる(t *testing.T) {
	t.Parallel()

	far := consts.Coord[consts.Tile]{X: 25, Y: 20}
	north, ok := render3d.NewProjector(defaultView, playerTile, screenW, screenH).TileCenter(far, 0)
	require.True(t, ok)

	turned := defaultView
	turned.Orient = 1
	rotated, ok := render3d.NewProjector(turned, playerTile, screenW, screenH).TileCenter(far, 0)
	require.True(t, ok)

	// 真北を向いていれば奥のタイルは画面中央の真上、45度回すと右へ寄る
	assert.InDelta(t, 480.0, float64(north.X), 0.05)
	assert.Greater(t, float64(rotated.X), float64(north.X)+100)
}

func TestProjector_四隅は北西から時計回りに並ぶ(t *testing.T) {
	t.Parallel()

	corners, ok := render3d.NewProjector(defaultView, playerTile, screenW, screenH).TileCorners(playerTile, 0)
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

	projector := render3d.NewProjector(defaultView, playerTile, screenW, screenH)
	tile := consts.Coord[consts.Tile]{X: 25, Y: 24}

	ground, ok := projector.TileCenter(tile, 0)
	require.True(t, ok)
	top, ok := projector.TileCenter(tile, render3d.WallHeight)
	require.True(t, ok)

	assert.Less(t, float64(top.Y), float64(ground.Y), "壁の天面は地面より上に写る")
}

func TestProjector_カメラ後方のタイルは投影できない(t *testing.T) {
	t.Parallel()

	// カメラはプレイヤーの南側にいる。そのさらに南は視線の裏側に回る
	_, ok := render3d.NewProjector(defaultView, playerTile, screenW, screenH).
		TileCenter(consts.Coord[consts.Tile]{X: 25, Y: 60}, 0)
	assert.False(t, ok)
}

func TestProjector_TileOnScreenは画面に掛かる升だけ通す(t *testing.T) {
	t.Parallel()

	projector := render3d.NewProjector(defaultView, playerTile, screenW, screenH)

	t.Run("足元のタイルは画面に掛かる", func(t *testing.T) {
		t.Parallel()
		assert.True(t, projector.TileOnScreen(playerTile, render3d.WallHeight))
	})
	t.Run("遠く横に外れたタイルは落とす", func(t *testing.T) {
		t.Parallel()
		// カメラの前方だが真横へ大きく外れるので画面矩形に掛からない
		assert.False(t, projector.TileOnScreen(consts.Coord[consts.Tile]{X: 200, Y: 25}, render3d.WallHeight))
	})
	t.Run("カメラ後方のタイルは落とす", func(t *testing.T) {
		t.Parallel()
		// 全隅が視線の裏側に回るので確実に落とせる
		assert.False(t, projector.TileOnScreen(consts.Coord[consts.Tile]{X: 25, Y: 60}, render3d.WallHeight))
	})
}

func TestProjector_TileOnScreenは大マップの大半を落とす(t *testing.T) {
	t.Parallel()

	// カリングが no-op に退化していないことのガード。100x100 マップの中央にプレイヤーを置くと、
	// 画面に掛かるのは視野内の一部だけで、大半は描画前に落ちる。総升は n^2 で増えるのに対し
	// 画面に掛かる升は視野で頭打ちなので、マップが大きいほど落とす割合は上がる。
	const n = 100
	center := consts.Coord[consts.Tile]{X: n / 2, Y: n / 2}
	p := render3d.NewProjector(defaultView, center, screenW, screenH)

	on := 0
	for y := range n {
		for x := range n {
			if p.TileOnScreen(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}, render3d.WallHeight) {
				on++
			}
		}
	}
	assert.Less(t, on, n*n/2, "100x100 の升のうち画面に掛かるのは半分未満")
}

func TestProjector_BillboardScaleは立て板の画面上の高さを返す(t *testing.T) {
	t.Parallel()

	projector := render3d.NewProjector(defaultView, playerTile, screenW, screenH)
	nearScale, ok := projector.BillboardScale(playerTile)
	require.True(t, ok)
	farScale, ok := projector.BillboardScale(consts.Coord[consts.Tile]{X: 25, Y: 20})
	require.True(t, ok)

	assert.Greater(t, nearScale, 0.0)
	assert.Greater(t, nearScale, farScale, "手前の立て板ほど大きく写る")
}

func TestProjector_Depthは奥ほど小さい(t *testing.T) {
	t.Parallel()

	projector := render3d.NewProjector(defaultView, playerTile, screenW, screenH)
	nearDepth := projector.Depth(render3d.Vec{X: 25.5, Z: 25.5})
	farDepth := projector.Depth(render3d.Vec{X: 25.5, Z: 20.5})

	assert.Less(t, farDepth, nearDepth, "奥のタイルほど奥行きが小さい")
}

func TestCameraYaw_Orientから水平角を導出する(t *testing.T) {
	t.Parallel()

	// 水平角を別フィールドに持たせず Orient から導くので、2箇所に分かれてずれることがない
	assert.InDelta(t, 0.0, gc.Camera{Orient: 0}.Yaw(), 1e-9)
	assert.InDelta(t, 0.7853981633974483, gc.Camera{Orient: 1}.Yaw(), 1e-9)
	assert.InDelta(t, 5.497787143782138, gc.Camera{Orient: 7}.Yaw(), 1e-9)
}
