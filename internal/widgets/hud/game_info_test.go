package hud

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOffsetTriangle_全頂点を同じ量だけ平行移動する(t *testing.T) {
	t.Parallel()

	pts := [3][2]float32{{0, 0}, {10, 0}, {5, 10}}

	got := offsetTriangle(pts, 3, -2)

	assert.Equal(t, [3][2]float32{{3, -2}, {13, -2}, {8, 8}}, got)
}

func TestOffsetTriangle_オフセットがゼロなら座標は変わらない(t *testing.T) {
	t.Parallel()

	pts := [3][2]float32{{1, 2}, {3, 4}, {5, 6}}

	got := offsetTriangle(pts, 0, 0)

	assert.Equal(t, pts, got)
}

func TestLerpTempColor(t *testing.T) {
	t.Parallel()

	a := color.RGBA{R: 0, G: 0, B: 0, A: 100}
	b := color.RGBA{R: 100, G: 200, B: 50, A: 50}

	t.Run("tが0ならaをそのまま返すが不透明度は255になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 0, G: 0, B: 0, A: 255}, lerpTempColor(a, b, 0))
	})

	t.Run("tが1ならbをそのまま返すが不透明度は255になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 100, G: 200, B: 50, A: 255}, lerpTempColor(a, b, 1))
	})

	t.Run("tが中間なら各チャンネルを線形補間する", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 50, G: 100, B: 25, A: 255}, lerpTempColor(a, b, 0.5))
	})
}

func TestBodyTempFillColor(t *testing.T) {
	t.Parallel()

	t.Run("平熱ちょうどは中立色になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 235, G: 235, B: 235, A: 255}, bodyTempFillColor(0.5))
	})

	t.Run("最も冷えていると青一色になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 40, G: 90, B: 230, A: 255}, bodyTempFillColor(0))
	})

	t.Run("最も火照っていると赤一色になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 230, G: 50, B: 40, A: 255}, bodyTempFillColor(1))
	})

	t.Run("冷え側の中間は中立色と青の中間になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 137, G: 162, B: 232, A: 255}, bodyTempFillColor(0.25))
	})

	t.Run("火照り側の中間は中立色と赤の中間になる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, color.RGBA{R: 232, G: 142, B: 137, A: 255}, bodyTempFillColor(0.75))
	})
}
