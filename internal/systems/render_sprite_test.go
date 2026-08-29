package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// 期待値の根拠。testutil.InitTestWorld が画面を 960x720 に固定し、consts.TileSize は 32。
// scale=1 では halfW=480 halfH=360 で、maxX=480/32+margin、maxY=360/32+margin になる。
// margin=2 なら maxX=17 maxY=13 で、min は原点対称に符号反転する。scale=2 では画面幅を半分に見るので
// halfW=240 halfH=180 になり、カメラ位置 320 を足して割ると 2,17,4,15 になる。
func TestViewportTileBounds(t *testing.T) {
	t.Parallel()

	t.Run("カメラがnilのときは原点基準で計算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		minX, maxX, minY, maxY := viewportTileBounds(world, 2, nil)

		assert.Equal(t, -17, minX)
		assert.Equal(t, 17, maxX)
		assert.Equal(t, -13, minY)
		assert.Equal(t, 13, maxY)
	})

	t.Run("カメラの位置とスケールを考慮する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		camera := &gc.Camera{Scale: 2.0, Pos: consts.Coord[consts.WorldPixel]{X: 320, Y: 320}}

		minX, maxX, minY, maxY := viewportTileBounds(world, 0, camera)

		assert.Equal(t, 2, minX)
		assert.Equal(t, 17, maxX)
		assert.Equal(t, 4, minY)
		assert.Equal(t, 15, maxY)
	})

	t.Run("スケールが0以下なら1.0として扱う", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		camera := &gc.Camera{Scale: 0}

		minX, maxX, minY, maxY := viewportTileBounds(world, 2, camera)

		assert.Equal(t, -17, minX)
		assert.Equal(t, 17, maxX)
		assert.Equal(t, -13, minY)
		assert.Equal(t, 13, maxY)
	})
}

func TestInViewport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		grid *gc.GridElement
		want bool
	}{
		{
			name: "範囲内",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}},
			want: true,
		},
		{
			name: "X下限ちょうどは含む",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 5}},
			want: true,
		},
		{
			name: "X上限ちょうどは含む",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 5}},
			want: true,
		},
		{
			name: "X下限未満は範囲外",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: -1, Y: 5}},
			want: false,
		},
		{
			name: "X上限超過は範囲外",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 5}},
			want: false,
		},
		{
			name: "Y下限未満は範囲外",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: -1}},
			want: false,
		},
		{
			name: "Y上限超過は範囲外",
			grid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 11}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := inViewport(tt.grid, 0, 10, 0, 10)
			assert.Equal(t, tt.want, got)
		})
	}
}
