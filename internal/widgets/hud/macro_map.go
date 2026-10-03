package hud

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
)

// MacroMap は HUD 右上のマクロ地図エリア。N キーで開く地形俯瞰と同じ内容を縮小して常時表示する。
type MacroMap struct {
	face    text.Face // 地形セルの記号とキューブに使うフォント
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
	// 右上へ固定サイズのパネルを置く。先頭0幅列で右寄せ、末尾 Grow で上寄せする
	panel := &macroPanelWidget{chrome: m.chrome, face: m.face, data: data}
	row := uicore.Row([]int{0, width}, uicore.NewGroup(), panel)
	items := []uicore.FlexItem{
		{W: row, Height: height},
		{Grow: true},
	}
	inner := image.Rect(0, theme.Space4, data.Screen.Width-theme.Space4, data.Screen.Height)
	uicore.FlexColumn(inner, items)
	drawFlexItems(cv, items)
}

// macroPanelWidget は右上のマクロ地図パネル。与えられた矩形に背景枠を敷き、帯全体を中央寄せで描く。
// 地図のセル座標計算は地図の本質なのでここに残し、パネルの画面内配置だけ FlexColumn/Row へ委ねる。
type macroPanelWidget struct {
	rect   image.Rectangle
	chrome Chrome
	face   text.Face
	data   MacroMapData
}

// Layout は uicore.Widget を満たす。
func (m *macroPanelWidget) Layout(r image.Rectangle) { m.rect = r }

// Draw は uicore.Widget を満たす。背景枠と、帯全体を収める正方セルの地図を描く。
func (m *macroPanelWidget) Draw(cv uicore.Canvas) {
	// 背景枠はメニュー枠と同じ共通 chrome に揃える
	m.chrome.Panel(cv, m.rect)

	rows := len(m.data.View.Cells)
	cols := len(m.data.View.Cells[0])
	// 帯全体がパネルに収まるようセル辺を決める。縦横で小さいほうに合わせて正方セルにする
	cellPx := m.rect.Dx() / cols
	if h := m.rect.Dy() / rows; h < cellPx {
		cellPx = h
	}
	if cellPx < 1 {
		cellPx = 1
	}
	// 地図をパネル内で中央寄せする余白
	offX := m.rect.Min.X + (m.rect.Dx()-cellPx*cols)/2
	offY := m.rect.Min.Y + (m.rect.Dy()-cellPx*rows)/2

	// 格子・道・キューブ・現在地を DrawMapGrid で描く。ミニマップは記号表示に閾値を持つ
	DrawMapGrid(cv, m.data.View, MapGridStyle{
		OriginX:      offX,
		OriginY:      offY,
		CellPx:       cellPx,
		MinGlyphPx:   m.data.Config.MinGlyphPx,
		GlyphFace:    m.face,
		PlayerFacing: m.data.PlayerFacing,
	})
}

// Children は uicore.Widget を満たす。子は持たない。
func (m *macroPanelWidget) Children() []uicore.Widget { return nil }

// drawCenteredGlyph はセルの中央に1文字を描く。字形の外接矩形を測って四辺の余白を揃えるので、
// フォントの行メトリクスでなく実際の字形がセル中央へ来る。小さいセルでも記号が上下へ偏らない。
func drawCenteredGlyph(cv uicore.Canvas, s string, face text.Face, x, y, cell int, col color.Color) {
	tw, th := uicore.MeasureText(s, face)
	cv.DrawText(image.Pt(x+(cell-tw)/2, y+(cell-th)/2), s, face, col)
}
