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
			// 道が通るチャンクは接続方角へ線分を引く。縮小地図でも街道の走りが読める
			if cell.Road.Any() {
				drawMacroRoad(cv, cx, cy, cellPx, cell.Road)
			}
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

// drawMacroRoad は全画面図 drawCellRoad と同じ道の線分を、縮小地図の整数座標で描く。太さは cell/4・
// 最低 1px とし、セルが小さくても道が消えないようにする。全画面図の cell/5・最低 2px とは縮尺ぶん差を付ける。
func drawMacroRoad(cv uicore.Canvas, cx, cy, cell int, road overworld.RoadDir) {
	t := max(cell/4, 1)
	half := t / 2
	ccx, ccy := cx+cell/2, cy+cell/2
	col := theme.OverworldMapRoad
	if road&overworld.RoadW != 0 {
		cv.FillRect(image.Rect(cx, ccy-half, ccx+half, ccy-half+t), col)
	}
	if road&overworld.RoadE != 0 {
		cv.FillRect(image.Rect(ccx-half, ccy-half, cx+cell, ccy-half+t), col)
	}
	if road&overworld.RoadN != 0 {
		cv.FillRect(image.Rect(ccx-half, cy, ccx-half+t, ccy+half), col)
	}
	if road&overworld.RoadS != 0 {
		cv.FillRect(image.Rect(ccx-half, ccy-half, ccx-half+t, cy+cell), col)
	}
}

// drawCenteredGlyph はセルの中央に1文字を描く。DrawText は左上基準なので、文字の寸法を測って
// セル内で中央へ寄せる。
func drawCenteredGlyph(cv uicore.Canvas, s string, face text.Face, x, y, cell int, col color.Color) {
	tw, th := uicore.MeasureText(s, face)
	cv.DrawText(image.Pt(x+(cell-tw)/2, y+(cell-th)/2), s, face, col)
}
