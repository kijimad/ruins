package hud

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/overworld"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// MapGridStyle はチャンク俯瞰グリッドを描くのに地図ごとに変わる値。寸法・フォント・現在地の向きを持つ。
// 色とキューブ・ポインタの意匠は地図で揃えるため DrawMapGrid が共通に決め、ここには持たせない。
type MapGridStyle struct {
	OriginX, OriginY int       // グリッド左上のピクセル
	CellPx           int       // 1セルの辺
	MinGlyphPx       int       // セル辺がこれ以上のとき種別記号を重ねる。0 なら常に描く
	GlyphFace        text.Face // セル記号とキューブに使うフォント
	MarkerFace       text.Face // 現在地ポインタに使うフォント。地図ごとにセル記号と同じか大きいかを選ぶ
	PlayerFacing     gc.Orient // 現在地ポインタの向き
}

// DrawMapGrid はチャンク俯瞰の格子・道・キューブ・現在地ポインタを cv へ描く。ミニマップと全画面図が
// 同じ見た目になるよう描画をここへ一本化する。見出しや凡例のような地図外の chrome は含めない。
func DrawMapGrid(cv uicore.Canvas, view overworld.MacroView, style MapGridStyle) {
	drawGlyph := style.CellPx >= style.MinGlyphPx
	for row := range view.Cells {
		for col, cell := range view.Cells[row] {
			// 未開放チャンクは描かず地を透かしてフォグにする。探索で徐々に開く
			if !cell.Discovered {
				continue
			}
			x := style.OriginX + col*style.CellPx
			y := style.OriginY + row*style.CellPx
			cv.FillRect(image.Rect(x, y, x+style.CellPx, y+style.CellPx), macroGlyphColor(cell.Glyph))
			if cell.Road.Any() {
				drawMapRoad(cv, x, y, style.CellPx, cell.Road)
			}
			if drawGlyph {
				drawCenteredGlyph(cv, string(cell.Glyph), style.GlyphFace, x, y, style.CellPx, theme.OverworldMapGlyphText)
			}
		}
	}
	// キューブのチャンク位置。現在地と重なるものは BuildMacroView が落とすのでここは描くだけ
	for _, c := range view.CubeCells {
		x := style.OriginX + int(c.X)*style.CellPx
		y := style.OriginY + int(c.Y)*style.CellPx
		drawCubeMarker(cv, style.GlyphFace, x, y, style.CellPx)
	}
	// 現在地。ナビのポインタをカメラ前方へ回して位置と向きを兼ねる。北=上なので指す向きが方角になる。
	// location-arrow は北東向きなので -π/4 で北へ補正し -yaw で前方へ回す
	if view.PlayerCell != nil {
		cx := style.OriginX + int(view.PlayerCell.X)*style.CellPx + style.CellPx/2
		cy := style.OriginY + int(view.PlayerCell.Y)*style.CellPx + style.CellPx/2
		angle := -style.PlayerFacing.Yaw() - math.Pi/4
		cv.DrawText(image.Pt(cx, cy), consts.IconLocationArrow, style.MarkerFace, theme.TextAccent, uicore.Rotated(angle))
	}
}

// drawMapRoad はチャンクセルを通る道を接続方角ごとに、セル中央から辺の中点へ細い矩形で引く。
// 太さはセル辺の1/5で最低1px。縮小地図でも街道の走りが読める。
func drawMapRoad(cv uicore.Canvas, x, y, cell int, road overworld.RoadDir) {
	t := max(cell/5, 1)
	half := t / 2
	ccx, ccy := x+cell/2, y+cell/2
	col := theme.OverworldMapRoad
	if road&overworld.RoadW != 0 {
		cv.FillRect(image.Rect(x, ccy-half, ccx+half, ccy-half+t), col)
	}
	if road&overworld.RoadE != 0 {
		cv.FillRect(image.Rect(ccx-half, ccy-half, x+cell, ccy-half+t), col)
	}
	if road&overworld.RoadN != 0 {
		cv.FillRect(image.Rect(ccx-half, y, ccx-half+t, ccy+half), col)
	}
	if road&overworld.RoadS != 0 {
		cv.FillRect(image.Rect(ccx-half, ccy-half, ccx-half+t, y+cell), col)
	}
}

// drawCubeMarker はキューブのチャンク位置を IconCube で描く。地形色に埋もれないよう暗い縁取りを
// 上下左右へ1pxずらして先に敷いてから本体を重ねる。
func drawCubeMarker(cv uicore.Canvas, face text.Face, x, y, cell int) {
	for _, off := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		drawCenteredGlyph(cv, consts.IconCube, face, x+off[0], y+off[1], cell, theme.OverworldMapCubeOutline)
	}
	drawCenteredGlyph(cv, consts.IconCube, face, x, y, cell, theme.OverworldMapCubeMarker)
}
