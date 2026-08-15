package systems

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestTileVisFactor はタイル描画情報から明るさ・可視・光源色を導く純関数を固定する。
// 3Dの Draw を通さず、Darkness の明るさ反映と可視/記憶/未探索の分岐を単体で検証する。
func TestTileVisFactor(t *testing.T) {
	t.Parallel()

	t.Run("可視はDarknessを明るさへ反映する", func(t *testing.T) {
		t.Parallel()
		bright, drawable, visible, light := tileVisFactor(TileRenderVisible{Darkness: 0.3})
		assert.InDelta(t, 0.7, bright, 1e-9, "bright = 1 - Darkness")
		assert.True(t, drawable)
		assert.True(t, visible)
		assert.Equal(t, [3]float64{1, 1, 1}, light, "無色の光源は白のまま")
	})

	t.Run("可視の光源色は最大成分で正規化して色味だけ返す", func(t *testing.T) {
		t.Parallel()
		_, _, _, light := tileVisFactor(TileRenderVisible{LightColor: color.RGBA{R: 255, G: 128, B: 0, A: 255}})
		assert.InDelta(t, 1.0, light[0], 1e-9)
		assert.InDelta(t, 128.0/255, light[1], 1e-9)
		assert.InDelta(t, 0.0, light[2], 1e-9)
	})

	t.Run("記憶は描画可だが可視ではない", func(t *testing.T) {
		t.Parallel()
		bright, drawable, visible, light := tileVisFactor(TileRenderRemembered{Darkness: 0.75})
		assert.InDelta(t, 0.25, bright, 1e-9)
		assert.True(t, drawable)
		assert.False(t, visible)
		assert.Equal(t, [3]float64{1, 1, 1}, light, "記憶は光源色を持たず白")
	})

	t.Run("未探索は描画しない", func(t *testing.T) {
		t.Parallel()
		bright, drawable, visible, _ := tileVisFactor(nil)
		assert.Zero(t, bright)
		assert.False(t, drawable)
		assert.False(t, visible)
	})
}
