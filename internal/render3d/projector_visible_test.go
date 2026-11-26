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
	for _, pitch := range []float64{gc.CameraMinPitch, gc.CameraDefaultPitch, gc.CameraMaxPitch} {
		for _, dist := range []float64{gc.CameraMinDist, gc.CameraDefaultDist, gc.CameraMaxDist} {
			for orient := range gc.Orient(gc.CameraOrientCount) {
				t.Run(fmt.Sprintf("pitch=%.2f/dist=%.0f/orient=%d", pitch, dist, orient), func(t *testing.T) {
					t.Parallel()
					p := NewProjector(gc.Camera{Orient: orient, Pitch: pitch, Dist: dist}, center, 960, 720)
					missed := 0
					for y := consts.Tile(-reach); y <= reach; y++ {
						for x := consts.Tile(-reach); x <= reach; x++ {
							c := consts.Coord[consts.Tile]{X: x, Y: y}
							if !allCornersInFront(p, c) || !p.boxOnScreen(c, maxStandHeight) {
								continue
							}
							if !p.inVisibleRect(c) {
								missed++
							}
						}
					}
					assert.Equal(t, 0, missed)
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
