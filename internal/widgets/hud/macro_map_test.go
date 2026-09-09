package hud

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMacroMap(t *testing.T) *MacroMap {
	t.Helper()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	return NewMacroMap(nil, NewChrome(res))
}

func TestMacroMap_Draw_無効なら何も描かない(t *testing.T) {
	t.Parallel()
	m := newTestMacroMap(t)
	m.enabled = false
	cv := &fakeCanvas{}

	m.Draw(cv, MacroMapData{
		Config: MacroMapConfig{Width: 150, Height: 150},
		Screen: ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, 0, cv.nineSlices)
	assert.Empty(t, cv.texts)
	assert.Empty(t, cv.fillRects)
}

func TestMacroMap_Draw_帯がなければ地図を出さない(t *testing.T) {
	t.Parallel()
	m := newTestMacroMap(t)
	cv := &fakeCanvas{}

	m.Draw(cv, MacroMapData{
		HasBand: false,
		Config:  MacroMapConfig{Width: 150, Height: 150},
		Screen:  ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, 0, cv.nineSlices, "帯が無ければ背景パネルも出さない")
	assert.Empty(t, cv.texts)
	assert.Empty(t, cv.fillRects)
}

func TestMacroMap_Draw_サイズ0なら背景パネルを描かない(t *testing.T) {
	t.Parallel()
	m := newTestMacroMap(t)
	cv := &fakeCanvas{}

	m.Draw(cv, MacroMapData{
		HasBand: true,
		Config:  MacroMapConfig{Width: 0, Height: 0},
		Screen:  ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, 0, cv.nineSlices, "サイズ0のときは背景を敷かない")
}

func TestMacroMap_Draw_開放セルを塗り未開放は伏せる(t *testing.T) {
	t.Parallel()
	m := newTestMacroMap(t)
	cv := &fakeCanvas{}

	// 開放済み1セルと未開放1セルを並べる。MinGlyphPx を大きくして記号描画は避ける
	view := overworld.MacroView{
		Cells: [][]overworld.MacroCell{{
			{Glyph: '.', Discovered: true},
			{Glyph: '.', Discovered: false},
		}},
		PlayerCell: consts.Coord[consts.Chunk]{X: -1},
	}
	m.Draw(cv, MacroMapData{
		HasBand: true,
		View:    view,
		Config:  MacroMapConfig{Width: 150, Height: 150, MinGlyphPx: 999},
		Screen:  ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, 1, cv.nineSlices, "背景パネルを描く")
	assert.Len(t, cv.fillRects, 1, "開放済みセルだけ塗り、未開放は伏せる")
}
