package hud

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMacroMap(t *testing.T) *MacroMap {
	t.Helper()
	// フェイスは nil でよい。fakeCanvas は DrawText を記録するだけで実描画しないため、フェイスに
	// 触れない。本番の EbitenCanvas には loader 由来の非 nil フェイスが渡る
	return NewMacroMap(nil, Chrome{})
}

func TestGlyphColor_全ての凡例記号に色が割り当てられている(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	for _, g := range overworld.LegendGlyphs(raws) {
		c, ok := overworld.GlyphColor(raws, g.Label)
		assert.Truef(t, ok, "凡例記号 %s(%c) に色がある", g.Name, g.Label)
		assert.NotEqualf(t, color.RGBA{}, c, "凡例記号 %s(%c) の色が透明黒でない", g.Name, g.Label)
	}
}

func TestGlyphColor_未知の文字は対応なし(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	_, ok := overworld.GlyphColor(raws, 'Z')
	assert.False(t, ok, "未知の文字は対応なしを返す")
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

	assert.Equal(t, 0, cv.roundedFills)
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

	assert.Equal(t, 0, cv.roundedFills, "帯が無ければ背景パネルも出さない")
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

	assert.Equal(t, 0, cv.roundedFills, "サイズ0のときは背景を敷かない")
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

	assert.Equal(t, 1, cv.roundedFills, "背景パネルを描く")
	assert.Len(t, cv.fillRects, 1, "開放済みセルだけ塗り、未開放は伏せる")
}

func TestMacroMap_Draw_現在地は三角ポインタで描く(t *testing.T) {
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

	assert.Len(t, cv.triangles, 2, "現在地は上下対称の二色針(前方金・後方シルバー)で示す")
	assert.Empty(t, cv.strokeRects, "四角枠の現在地マーカーは描かない")
}

func TestDrawMapGrid_現在地ポインタは向きへtipを向ける(t *testing.T) {
	t.Parallel()

	// PlayerCell(0,0)・cell=20 なので中央は (10,10)。tip はローカル (0,-0.42*cell) を向きだけ回した位置
	const cell = 20
	cx, cy, fwd := 10.0, 10.0, playerMarkerTip*float64(cell)
	cases := []struct {
		name               string
		facing             gc.Orient
		wantTipX, wantTipY float64
	}{
		{"北", 0, cx, cy - fwd},
		{"東", 2, cx + fwd, cy},
		{"南", 4, cx, cy + fwd},
		{"西", 6, cx - fwd, cy},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cv := &fakeCanvas{}
			view := overworld.MacroView{
				Cells:      [][]overworld.MacroCell{{{Glyph: '.', Discovered: true}}},
				PlayerCell: &consts.Coord[consts.Chunk]{X: 0, Y: 0},
			}
			DrawMapGrid(cv, view, MapGridStyle{CellPx: cell, MinGlyphPx: 999, PlayerFacing: tc.facing})

			require.Len(t, cv.triangles, 2, "前方と後方の2枚を描く")
			tip := cv.triangles[0][0] // 前方の三角(金)の頂点0が前方の針先
			assert.InDelta(t, tc.wantTipX, float64(tip[0]), 1e-4)
			assert.InDelta(t, tc.wantTipY, float64(tip[1]), 1e-4)
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

	assert.Empty(t, cv.triangles, "プレイヤー不在なら現在地ポインタを描かない")
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
	raws := testutil.InitTestWorld(t).Resources.RawMaster

	// フェイスは nil でよい。fakeCanvas は描画命令を記録するだけで実描画しない
	DrawMapLegend(cv, raws, nil, nil, 100)

	glyphs := overworld.LegendGlyphs(raws)
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
