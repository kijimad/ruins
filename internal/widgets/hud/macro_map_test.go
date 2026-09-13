package hud

import (
	"math"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMacroMap(t *testing.T) *MacroMap {
	t.Helper()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	// フェイスは nil でよい。fakeCanvas は DrawText を記録するだけで実描画しないため、フェイスに
	// 触れない。本番の EbitenCanvas には loader 由来の非 nil フェイスが渡る
	return NewMacroMap(nil, NewChrome(res))
}

func TestMacroGlyphColor_全ての種別記号に色が割り当てられている(t *testing.T) {
	t.Parallel()

	fallback := macroGlyphColor('\x00') // 未知の文字の色
	for _, g := range overworld.PlaceGlyphs() {
		assert.NotEqualf(t, fallback, macroGlyphColor(g.Label), "地物 %s(%c) に固有色がある", g.Name, g.Label)
	}
	for _, g := range overworld.FacilityGlyphs() {
		assert.NotEqualf(t, fallback, macroGlyphColor(g.Label), "施設 %s(%c) に固有色がある", g.Name, g.Label)
	}
}

func TestMacroGlyphColor_未知の文字は灰色のフォールバック(t *testing.T) {
	t.Parallel()

	assert.Equal(t, macroGlyphColor('\x00'), macroGlyphColor('Z'), "未知の文字は同じフォールバック色になる")
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

	// 現在地は location-arrow を1つだけ描く。記号で引き当てて基準点と回転角を確かめる
	var pointer *textCall
	for i := range cv.texts {
		if cv.texts[i].str == consts.IconLocationArrow {
			require.Nil(t, pointer, "現在地ポインタは1つだけ")
			pointer = &cv.texts[i]
		}
	}
	require.NotNil(t, pointer, "現在地ポインタを描く")
	assert.Equal(t, uicore.AnchorCenter, pointer.anchor, "ポインタは中央基準で描く")
	// 北向き PlayerFacing=0 は Yaw=0。location-arrow は北東向きなので -π/4 で北へ補正する
	assert.InDelta(t, -math.Pi/4, pointer.angle, 1e-9, "北向きのポインタは北東基準から -π/4 回す")
	assert.Empty(t, cv.strokeRects, "四角枠の現在地マーカーは描かない")
}

func TestDrawMapGrid_現在地ポインタの回転角は向きで決まる(t *testing.T) {
	t.Parallel()

	// angle = -Yaw - π/4。北 Orient0 は Yaw0、南 Orient4 は Yaw=π
	cases := []struct {
		name   string
		facing gc.Orient
		want   float64
	}{
		{"北", 0, -math.Pi / 4},
		{"東", 2, -math.Pi/2 - math.Pi/4},
		{"南", 4, -math.Pi - math.Pi/4},
		{"西", 6, -3*math.Pi/2 - math.Pi/4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cv := &fakeCanvas{}
			view := overworld.MacroView{
				Cells:      [][]overworld.MacroCell{{{Glyph: '.', Discovered: true}}},
				PlayerCell: &consts.Coord[consts.Chunk]{X: 0, Y: 0},
			}
			DrawMapGrid(cv, view, MapGridStyle{CellPx: 20, MinGlyphPx: 999, PlayerFacing: tc.facing})

			var pointer *textCall
			for i := range cv.texts {
				if cv.texts[i].str == consts.IconLocationArrow {
					pointer = &cv.texts[i]
				}
			}
			require.NotNil(t, pointer, "現在地ポインタを描く")
			assert.InDelta(t, tc.want, pointer.angle, 1e-9)
		})
	}
}

func TestDrawMapGrid_プレイヤー不在なら現在地ポインタを描かない(t *testing.T) {
	t.Parallel()
	cv := &fakeCanvas{}

	// PlayerCell を nil のままにする
	view := overworld.MacroView{
		Cells: [][]overworld.MacroCell{{{Glyph: '.', Discovered: true}}},
	}
	DrawMapGrid(cv, view, MapGridStyle{CellPx: 20, MinGlyphPx: 999})

	for _, tc := range cv.texts {
		assert.NotEqual(t, consts.IconLocationArrow, tc.str, "プレイヤー不在なら現在地ポインタを描かない")
	}
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

func TestDrawMapLegend_種別ごとに色見本と名前を描く(t *testing.T) {
	t.Parallel()
	cv := &fakeCanvas{}

	// フェイスは nil でよい。fakeCanvas は描画命令を記録するだけで実描画しない
	DrawMapLegend(cv, nil, nil, 100)

	glyphs := overworld.LegendGlyphs()
	require.NotEmpty(t, glyphs)
	assert.Len(t, cv.fillRects, len(glyphs), "種別ごとに色見本を1つ塗る")
	// 種別ごとに記号1つと名前1つ、末尾に閉じ方の案内を描く
	assert.Len(t, cv.texts, len(glyphs)*2+1)

	var hasClose bool
	for _, tc := range cv.texts {
		if tc.str == "N / Esc to close" {
			hasClose = true
		}
	}
	assert.True(t, hasClose, "閉じ方の案内を描く")
}

func TestDrawMapGrid_極小セルでも道が消えない(t *testing.T) {
	t.Parallel()
	cv := &fakeCanvas{}

	// t = max(cell/5, 1) のクランプで、cell=3 でも太さ1pxの道が残る
	view := overworld.MacroView{
		Cells: [][]overworld.MacroCell{{{
			Glyph:      '.',
			Discovered: true,
			Road:       overworld.RoadN | overworld.RoadS | overworld.RoadE | overworld.RoadW,
		}}},
	}
	DrawMapGrid(cv, view, MapGridStyle{CellPx: 3, MinGlyphPx: 999})

	assert.Len(t, cv.fillRects, 5, "極小セルでも地色1つと4方角の道4つを描く")
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
