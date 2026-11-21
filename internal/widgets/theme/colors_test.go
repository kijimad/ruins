package theme_test

import (
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
)

func TestLerpColor(t *testing.T) {
	t.Parallel()

	a := color.RGBA{R: 0, G: 0, B: 0, A: 100}
	b := color.RGBA{R: 100, G: 200, B: 50, A: 50}

	t.Run("tが0ならaをそのまま返すが不透明度は255になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 0, G: 0, B: 0, A: 255}, theme.LerpColor(a, b, 0))
	})

	t.Run("tが1ならbをそのまま返すが不透明度は255になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 100, G: 200, B: 50, A: 255}, theme.LerpColor(a, b, 1))
	})

	t.Run("tが中間なら各チャンネルを線形補間する", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 50, G: 100, B: 25, A: 255}, theme.LerpColor(a, b, 0.5))
	})
}

func TestHealthGaugeColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		ratio float64
		want  color.RGBA
	}{
		{name: "満タンは満タンの色", ratio: 1, want: theme.LerpColor(theme.HUDHealthFull, theme.HUDHealthHalf, 0)},
		{name: "半分ちょうどは半分の色", ratio: 0.5, want: theme.LerpColor(theme.HUDHealthEmpty, theme.HUDHealthHalf, 1)},
		{name: "半分より上は満タンと半分の間", ratio: 0.75, want: theme.LerpColor(theme.HUDHealthFull, theme.HUDHealthHalf, 0.5)},
		{name: "半分より下は空と半分の間", ratio: 0.25, want: theme.LerpColor(theme.HUDHealthEmpty, theme.HUDHealthHalf, 0.5)},
		{name: "空は空の色", ratio: 0, want: theme.LerpColor(theme.HUDHealthEmpty, theme.HUDHealthHalf, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, theme.HealthGaugeColor(tt.ratio))
		})
	}
}
