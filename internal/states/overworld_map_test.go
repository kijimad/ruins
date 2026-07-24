package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/stretchr/testify/assert"
)

func TestGlyphColor_全ての種別記号に色が割り当てられている(t *testing.T) {
	t.Parallel()

	// 地物レベルの記号
	features := []rune{
		overworld.GlyphField, overworld.GlyphRoad, overworld.GlyphVillage,
		overworld.GlyphHamlet, overworld.GlyphRuin, overworld.GlyphPOI,
	}
	fallback := glyphColor('\x00') // 未知の文字の色
	for _, r := range features {
		assert.NotEqualf(t, fallback, glyphColor(r), "地物記号 %c に固有色がある", r)
	}
	// 施設種別
	for _, g := range overworld.FacilityGlyphs() {
		assert.NotEqualf(t, fallback, glyphColor(g.Label), "施設 %s(%c) に固有色がある", g.Name, g.Label)
	}
}

func TestDownsampleRunes_地物を原野より優先して残す(t *testing.T) {
	t.Parallel()

	// 4x4 の原野に1つだけ遺跡入口を置く。2倍縮約でその存在が残ることを確かめる
	full := make([][]rune, 4)
	for y := range full {
		full[y] = []rune{overworld.GlyphField, overworld.GlyphField, overworld.GlyphField, overworld.GlyphField}
	}
	full[3][3] = overworld.GlyphRuin

	out := downsampleRunes(full, 2)
	assert.Len(t, out, 2, "縦は2行に縮む")
	assert.Len(t, out[0], 2, "横は2列に縮む")
	assert.Equal(t, overworld.GlyphRuin, out[1][1], "地物は原野に埋もれず代表として残る")
	assert.Equal(t, overworld.GlyphField, out[0][0], "地物の無いブロックは原野のまま")
}

func TestCeilDiv(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, ceilDiv(5, 2), "5/2 は切り上げて3")
	assert.Equal(t, 2, ceilDiv(4, 2), "割り切れるときはそのまま")
	assert.Equal(t, 1, ceilDiv(1, 5), "小さい分子でも最低1")
}
