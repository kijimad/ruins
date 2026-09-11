package hud

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/overworld"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMacroGlyphColor は既知の記号が overworld の色定義を引き、未知の記号が灰色へ
// フォールバックすることを検証する。
func TestMacroGlyphColor(t *testing.T) {
	t.Parallel()

	// 既知の記号は overworld の色定義を引く
	glyphs := overworld.LegendGlyphs()
	require.NotEmpty(t, glyphs)
	known := glyphs[0].Label
	want, ok := overworld.GlyphColor(known)
	require.True(t, ok, "凡例の記号は既知の前提")
	assert.Equal(t, want, macroGlyphColor(known), "既知の記号は定義色を返す")

	// 未知の記号は灰色にフォールバックする
	const unknown = '￿'
	_, ok = overworld.GlyphColor(unknown)
	require.False(t, ok, "この記号は未知の前提")
	assert.Equal(t, theme.OverworldMapUnknownGlyph, macroGlyphColor(unknown), "未知の記号は灰色")
}

// TestMacroMap_Draw_大きいセルは記号と現在地を重ねる はセルが MinGlyphPx 以上のとき
// 種別記号を重ね、現在地セルに白枠を描くことを検証する。記号描画は実 face が要る。
func TestMacroMap_Draw_大きいセルは記号と現在地を重ねる(t *testing.T) {
	t.Parallel()

	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	m := NewMacroMap(res.Text.BodyFace, NewChrome(res))
	cv := &fakeCanvas{}

	// 開放済み1セル。MinGlyphPx を小さくして記号を重ねる。現在地は 0,0
	view := overworld.MacroView{
		Cells: [][]overworld.MacroCell{{
			{Glyph: '.', Discovered: true},
		}},
		PlayerCell: consts.Coord[consts.Chunk]{X: 0, Y: 0},
		CubeCells:  []consts.Coord[consts.Chunk]{{X: 0, Y: 0}},
	}
	m.Draw(cv, MacroMapData{
		HasBand: true,
		View:    view,
		Config:  MacroMapConfig{Width: 150, Height: 150, MinGlyphPx: 1},
		Screen:  ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, 1, cv.nineSlices, "背景パネルを描く")
	assert.Len(t, cv.fillRects, 2, "開放セルの塗りとキューブマーカーを描く")
	require.Len(t, cv.texts, 1, "記号を1つ重ねる")
	assert.Equal(t, ".", cv.texts[0].str, "セルの記号を描く")
	assert.Len(t, cv.strokeRects, 1, "現在地マーカーの白枠を描く")
}
