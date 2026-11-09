package hud

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

// TestTileFrame_四辺を線で結ぶ は、投影済みの四隅をそのまま結んだ輪郭が描かれ、内側は塗らないことを確認する。
func TestTileFrame_四辺を線で結ぶ(t *testing.T) {
	t.Parallel()

	screen := ebiten.NewImage(14, 14)
	corners := [4]consts.Coord[consts.ScreenPixel]{
		{X: 2, Y: 2},   // 北西
		{X: 10, Y: 2},  // 北東
		{X: 10, Y: 10}, // 南東
		{X: 2, Y: 10},  // 南西
	}
	TileFrame(screen, corners, 2, color.RGBA{R: 255, A: 255})

	assertOpaque := func(x, y int, msg string) {
		_, _, _, a := screen.At(x, y).RGBA()
		assert.NotZero(t, a, msg)
	}
	assertOpaque(6, 2, "北辺に線を描く")
	assertOpaque(10, 6, "東辺に線を描く")
	assertOpaque(6, 10, "南辺に線を描く")
	assertOpaque(2, 6, "西辺に線を描く")

	_, _, _, centerA := screen.At(6, 6).RGBA()
	assert.Zero(t, centerA, "内側は塗らず輪郭だけを描く")
}

func TestScaleAlpha(t *testing.T) {
	t.Parallel()

	base := color.RGBA{R: 10, G: 20, B: 30, A: 200}

	tests := []struct {
		name  string
		alpha float64
		want  uint8
	}{
		{"係数1なら不透明度は変わらない", 1.0, 200},
		{"係数0.5なら不透明度は半分になる", 0.5, 100},
		{"係数0なら不透明度は0になる", 0.0, 0},
		{"負の係数は0にクランプされる", -0.5, 0},
		{"1を超える係数は1にクランプされる", 1.5, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ScaleAlpha(base, tt.alpha)
			assert.Equal(t, tt.want, got.A)
			assert.Equal(t, base.R, got.R)
			assert.Equal(t, base.G, got.G)
			assert.Equal(t, base.B, got.B)
		})
	}
}
