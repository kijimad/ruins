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
	face    text.Face
	chrome  Chrome
	enabled bool
}

// NewMacroMap は新しい HUD マクロ地図を作成する。
func NewMacroMap(face text.Face, chrome Chrome) *MacroMap {
	return &MacroMap{face: face, chrome: chrome, enabled: true}
}

// Update はマクロ地図を更新する。描画データは毎フレーム抽出されるので保持状態は持たない。
func (m *MacroMap) Update(_ w.World) {}

// Draw は帯全体の地形俯瞰をパネルへ縮小して描く。オーバーワールド外は No Data を出す。
func (m *MacroMap) Draw(cv uicore.Canvas, data MacroMapData) {
	if !m.enabled {
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

	// オーバーワールド外は帯が無いので No Data を出す
	if !data.HasBand || len(data.View.Cells) == 0 || len(data.View.Cells[0]) == 0 {
		drawOutlinedText(cv, "No Data", m.face, image.Pt(x0+50, y0+70), theme.TextPrimary)
		return
	}

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

	// 地形色のセルを敷く。セルが十分大きいときだけ種別記号を重ねる。
	// 未開放のチャンクは描かず、パネル背景のまま伏せてフォグにする。探索で徐々に開く
	drawGlyph := cellPx >= data.Config.MinGlyphPx
	for row := range data.View.Cells {
		for col, cell := range data.View.Cells[row] {
			if !cell.Discovered {
				continue
			}
			cx := offX + col*cellPx
			cy := offY + row*cellPx
			cv.FillRect(image.Rect(cx, cy, cx+cellPx, cy+cellPx), macroGlyphColor(cell.Glyph))
			if drawGlyph {
				drawCenteredGlyph(cv, string(cell.Glyph), m.face, cx, cy, cellPx, theme.OverworldMapGlyphText)
			}
		}
	}

	// キューブマーカー。地形色を残すためセルより内側に水色の四角を置く
	for _, c := range data.View.CubeCells {
		cx := offX + int(c.X)*cellPx
		cy := offY + int(c.Y)*cellPx
		inset := cellPx / 4
		cv.FillRect(image.Rect(cx+inset, cy+inset, cx+cellPx-inset, cy+cellPx-inset), theme.OverworldMapCubeMarker)
	}

	// 現在地マーカー。白枠でセルを囲む
	if data.View.PlayerCell.X >= 0 {
		px := offX + int(data.View.PlayerCell.X)*cellPx
		py := offY + int(data.View.PlayerCell.Y)*cellPx
		cv.StrokeRect(image.Rect(px, py, px+cellPx, py+cellPx), 1, theme.OverworldMapPlayerMarker)
	}
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
