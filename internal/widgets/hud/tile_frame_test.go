package hud

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
