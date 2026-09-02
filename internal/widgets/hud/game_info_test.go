package hud

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLerpColor(t *testing.T) {
	t.Parallel()

	a := color.RGBA{R: 0, G: 0, B: 0, A: 100}
	b := color.RGBA{R: 100, G: 200, B: 50, A: 50}

	t.Run("tが0ならaをそのまま返すが不透明度は255になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 0, G: 0, B: 0, A: 255}, lerpColor(a, b, 0))
	})

	t.Run("tが1ならbをそのまま返すが不透明度は255になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 100, G: 200, B: 50, A: 255}, lerpColor(a, b, 1))
	})

	t.Run("tが中間なら各チャンネルを線形補間する", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 50, G: 100, B: 25, A: 255}, lerpColor(a, b, 0.5))
	})
}
