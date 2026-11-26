package render3d

import (
	"fmt"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestVisibleTileRect_カメラの前にあって画面に掛かるタイルは矩形に入る(t *testing.T) {
	t.Parallel()
	const reach = 210
	center := consts.Coord[consts.Tile]{}
	// 0.44 と 0.47 は、視錐台の上の辺が水平をまたぐ境目の両側
	for _, pitch := range []float64{gc.CameraMinPitch, 0.44, 0.47, gc.CameraDefaultPitch, gc.CameraMaxPitch} {
		for _, dist := range []float64{gc.CameraMinDist, gc.CameraDefaultDist, gc.CameraMaxDist} {
			for orient := range gc.Orient(gc.CameraOrientCount) {
				t.Run(fmt.Sprintf("pitch=%.2f/dist=%.0f/orient=%d", pitch, dist, orient), func(t *testing.T) {
					t.Parallel()
					p := NewProjector(gc.Camera{Orient: orient, Pitch: pitch, Dist: dist}, center, 960, 720)
					var missed []consts.Coord[consts.Tile]
					for y := consts.Tile(-reach); y <= reach; y++ {
						for x := consts.Tile(-reach); x <= reach; x++ {
							c := consts.Coord[consts.Tile]{X: x, Y: y}
							if !allCornersInFront(p, c) || !p.boxOnScreen(c, maxStandHeight) {
								continue
							}
							if !p.inVisibleRect(c) {
								missed = append(missed, c)
							}
						}
					}
					assert.Empty(t, missed)
				})
			}
		}
	}
}

func TestVisibleTileRect_既定の視点では帯の大半を矩形の外に落とす(t *testing.T) {
	t.Parallel()
	p := NewProjector(gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}, consts.Coord[consts.Tile]{}, 960, 720)
	w := int(p.visMax.X-p.visMin.X) + 1
	h := int(p.visMax.Y-p.visMin.Y) + 1
	const band = 15 * 24
	assert.Less(t, w*h, band*band/4, "15×15 の帯の4分の1より小さい矩形に絞る")
}

func allCornersInFront(p Projector, c consts.Coord[consts.Tile]) bool {
	fx, fz := float64(c.X), float64(c.Y)
	for _, y := range [2]float64{0, maxStandHeight} {
		for _, v := range [4]Vec{{fx, y, fz}, {fx + 1, y, fz}, {fx + 1, y, fz + 1}, {fx, y, fz + 1}} {
			if _, ok := p.Point(v); !ok {
				return false
			}
		}
	}
	return true
}

func TestVisibleTileRect_視錐台が層と重ならなければどのタイルも入らない(t *testing.T) {
	t.Parallel()
	eye := Vec{0, 50, 0}
	visMin, visMax := visibleTileRect(eye, Vec{0, 100, -1}, 960.0/720.0)
	p := Projector{visMin: visMin, visMax: visMax}
	for _, c := range []consts.Coord[consts.Tile]{{}, {X: -1, Y: -1}, {X: 1, Y: 1}, {X: 0, Y: -100}} {
		assert.False(t, p.inVisibleRect(c), "%v", c)
	}
}
