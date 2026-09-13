package hud

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/overworld"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
)

// MacroMap は HUD 右上のマクロ地図エリア。N キーで開く地形俯瞰と同じ内容を縮小して常時表示する。
type MacroMap struct {
	face    text.Face // 地形セルの記号と現在地ポインタに使うフォント
	chrome  Chrome
	enabled bool
}

// NewMacroMap は新しい HUD マクロ地図を作成する。
func NewMacroMap(face text.Face, chrome Chrome) *MacroMap {
	return &MacroMap{face: face, chrome: chrome, enabled: true}
}

// Update はマクロ地図を更新する。描画データは毎フレーム抽出されるので保持状態は持たない。
func (m *MacroMap) Update(_ w.World) {}

// Draw はプレイヤー近傍の地形俯瞰をパネルへ描く。オーバーワールド外は地図を出さない。
func (m *MacroMap) Draw(cv uicore.Canvas, data MacroMapData) {
	if !m.enabled {
		return
	}
	// オーバーワールド外は帯が無いので、地図パネルごと出さない
	if !data.HasBand || len(data.View.Cells) == 0 || len(data.View.Cells[0]) == 0 {
		return
	}
	width, height := data.Config.Width, data.Config.Height
	if width <= 0 || height <= 0 {
		return
	}
	x0 := data.Screen.Width - width - theme.Space4
	y0 := theme.Space4
	// 背景枠はメニュー枠と同じ共通 chrome に揃える
	m.chrome.Panel(cv, image.Rect(x0, y0, x0+width, y0+height))

	rows := len(data.View.Cells)
	cols := len(data.View.Cells[0])
	// 帯全体がパネルに収まるようセル辺を決める。縦横で小さいほうに合わせて正方セルにする
	cellPx := width / cols
	if h := height / rows; h < cellPx {
		cellPx = h
	}
	if cellPx < 1 {
		cellPx = 1
	}
	// 地図をパネル内で中央寄せする余白
	offX := x0 + (width-cellPx*cols)/2
	offY := y0 + (height-cellPx*rows)/2

	// 格子・道・キューブ・現在地を DrawMapGrid で描く。ミニマップは記号表示に閾値を持つ
	DrawMapGrid(cv, data.View, MapGridStyle{
		OriginX:      offX,
		OriginY:      offY,
		CellPx:       cellPx,
		MinGlyphPx:   data.Config.MinGlyphPx,
		GlyphFace:    m.face,
		MarkerFace:   m.face,
		PlayerFacing: data.PlayerFacing,
	})
}

// macroGlyphColor は種別文字の色を返す。既知の記号は overworld の色定義を引き、未知は灰色にする。
func macroGlyphColor(r rune) color.RGBA {
	if c, ok := overworld.GlyphColor(r); ok {
		return c
	}
	return theme.OverworldMapUnknownGlyph
}

// drawCenteredGlyph はセルの中央に1文字を描く。DrawText は左上基準なので、文字の寸法を測って
// セル内で中央へ寄せる。
func drawCenteredGlyph(cv uicore.Canvas, s string, face text.Face, x, y, cell int, col color.Color) {
	tw, th := uicore.MeasureText(s, face)
	cv.DrawText(image.Pt(x+(cell-tw)/2, y+(cell-th)/2), s, face, col)
}
