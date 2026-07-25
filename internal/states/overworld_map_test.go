package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/stretchr/testify/assert"
)

func TestGlyphColor_全ての種別記号に色が割り当てられている(t *testing.T) {
	t.Parallel()

	features := []rune{
		overworld.GlyphField, overworld.GlyphVillage,
		overworld.GlyphHamlet, overworld.GlyphRuin, overworld.GlyphPOI,
	}
	fallback := glyphColor('\x00') // 未知の文字の色
	for _, r := range features {
		assert.NotEqualf(t, fallback, glyphColor(r), "地物記号 %c に固有色がある", r)
	}
	for _, g := range overworld.FacilityGlyphs() {
		assert.NotEqualf(t, fallback, glyphColor(g.Label), "施設 %s(%c) に固有色がある", g.Name, g.Label)
	}
}

func TestGlyphColor_未知の文字は灰色のフォールバック(t *testing.T) {
	t.Parallel()

	assert.Equal(t, glyphColor('\x00'), glyphColor('Z'), "未知の文字は同じフォールバック色になる")
}
