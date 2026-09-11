package hud

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

// TestTileFrame_台形の四隅を渡して描画しても落ちない は投影済みの四隅を線で結ぶ枠描画が
// 実 screen へ描いても落ちないことを検証する。台形でも四隅を順に結ぶだけなので、正方でない
// 四隅でも描けることを確認する。
func TestTileFrame_台形の四隅を渡して描画しても落ちない(t *testing.T) {
	t.Parallel()

	screen := ebiten.NewImage(64, 64)
	// 北西・北東・南東・南西。透視で潰れた台形を模す
	corners := [4]consts.Coord[consts.ScreenPixel]{
		{X: 10, Y: 10},
		{X: 54, Y: 12},
		{X: 50, Y: 54},
		{X: 14, Y: 50},
	}

	assert.NotPanics(t, func() {
		TileFrame(screen, corners, 2, color.RGBA{R: 255, A: 255})
	})
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
