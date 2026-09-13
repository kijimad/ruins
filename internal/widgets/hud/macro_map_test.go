package hud

import (
	"math"
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
	// フェイスは nil でよい。fakeCanvas は DrawText/DrawGlyphRotated を記録するだけで実描画しないため、
	// フェイスに触れない。本番の EbitenCanvas には loader 由来の非 nil フェイスが渡る
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

func TestMacroMap_Draw_現在地は回転ポインタで描く(t *testing.T) {
	t.Parallel()
	m := newTestMacroMap(t)
	cv := &fakeCanvas{}

	view := overworld.MacroView{
		Cells:      [][]overworld.MacroCell{{{Glyph: '.', Discovered: true}}},
		PlayerCell: &consts.Coord[consts.Chunk]{X: 0, Y: 0},
	}
	m.Draw(cv, MacroMapData{
		HasBand:      true,
		View:         view,
		PlayerFacing: 0,
		Config:       MacroMapConfig{Width: 150, Height: 150, MinGlyphPx: 999},
		Screen:       ScreenDimensions{Width: 1024, Height: 768},
	})

	assert.Equal(t, []string{consts.IconLocationArrow}, cv.rotatedGlyphs, "現在地はカメラ前方へ回したポインタ1つで示す")
	// 北向き PlayerFacing=0 は Yaw=0。location-arrow は北東向きなので -π/4 で北へ補正する
	require.Len(t, cv.rotatedAngles, 1)
	assert.InDelta(t, -math.Pi/4, cv.rotatedAngles[0], 1e-9, "北向きのポインタは北東基準から -π/4 回す")
	assert.Empty(t, cv.strokeRects, "四角枠の現在地マーカーは描かない")
}

func TestDrawMapGrid_道を持つセルは接続方角ごとに線分を描く(t *testing.T) {
	t.Parallel()
	cv := &fakeCanvas{}

	// 4方角すべてに繋がる道を持つ開放済み1セル。プレイヤー不在・キューブ無しにして道だけを数える
	view := overworld.MacroView{
		Cells: [][]overworld.MacroCell{{{
			Glyph:      '.',
			Discovered: true,
			Road:       overworld.RoadN | overworld.RoadS | overworld.RoadE | overworld.RoadW,
		}}},
	}
	DrawMapGrid(cv, view, MapGridStyle{CellPx: 20, MinGlyphPx: 999})

	// セルの地色1つと、接続方角4つの道で計5つの矩形を塗る
	assert.Len(t, cv.fillRects, 5, "地色1つと4方角の道4つを描く")
}

func TestDrawMapGrid_キューブは縁取り付きの記号で描く(t *testing.T) {
	t.Parallel()
	cv := &fakeCanvas{}

	view := overworld.MacroView{
		Cells:     [][]overworld.MacroCell{{{Glyph: '.', Discovered: true}}},
		CubeCells: []consts.Coord[consts.Chunk]{{X: 0, Y: 0}},
	}
	DrawMapGrid(cv, view, MapGridStyle{CellPx: 20, MinGlyphPx: 999})

	cubes := 0
	for _, tc := range cv.texts {
		if tc.str == consts.IconCube {
			cubes++
		}
	}
	assert.Equal(t, 5, cubes, "キューブは縁取り4つと本体1つで計5回描く")
}
