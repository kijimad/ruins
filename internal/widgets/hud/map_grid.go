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
	MinGlyphPx       int       // セル辺がこれ以上のときセル記号を重ねる。0 なら常に描く。キューブと現在地の印は閾値によらず常に描く
	GlyphFace        text.Face // セル記号とキューブに使うフォント
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
			cv.FillRect(image.Rect(x, y, x+style.CellPx, y+style.CellPx), macroGlyphColor(cell.Glyph), uicore.RectOptions{})
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
	// 現在地。上向きの三角形を向きだけ回して位置と向きを兼ねる
	if view.PlayerCell != nil {
		cx := float64(style.OriginX + int(view.PlayerCell.X)*style.CellPx + style.CellPx/2)
		cy := float64(style.OriginY + int(view.PlayerCell.Y)*style.CellPx + style.CellPx/2)
		drawPlayerMarker(cv, cx, cy, float64(style.CellPx), style.PlayerFacing)
	}
}

// 現在地三角形の各頂点のセル辺への比。縦横を近づけて縦長を避ける
const (
	playerMarkerTip  = 0.42 // tip の前方距離
	playerMarkerHalf = 0.30 // 底辺の半幅
	playerMarkerBack = 0.24 // 底辺の後方距離
)

// drawPlayerMarker は現在地の三角形を描く。無回転で北(上)を指す三角形を、中央を軸に向きだけ回す。
// tip が前方、底辺2点が後方。全体を尾の色で塗り、前方の鼻先を明るい色で重ねて向きを際立たせる。
func drawPlayerMarker(cv uicore.Canvas, cx, cy, cell float64, facing gc.Orient) {
	// 中央原点のローカル頂点。y は下向きなので前方(北)は負
	tip := [2]float64{0, -cell * playerMarkerTip}
	baseL := [2]float64{-cell * playerMarkerHalf, cell * playerMarkerBack}
	baseR := [2]float64{cell * playerMarkerHalf, cell * playerMarkerBack}
	// tip から底辺へ半分進んだ2点で前後を分け、前方だけ明るくする
	mid := func(a, b [2]float64) [2]float64 { return [2]float64{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2} }
	noseL, noseR := mid(tip, baseL), mid(tip, baseR)

	sin, cos := math.Sin(facing.Yaw()), math.Cos(facing.Yaw())
	// 標準の回転行列。y 下向きの画面座標では正の角度が時計回りになり、Orient の増加(北→東→南)と一致する
	rot := func(v [2]float64) [2]float32 {
		return [2]float32{float32(cx + v[0]*cos - v[1]*sin), float32(cy + v[0]*sin + v[1]*cos)}
	}
	cv.FillTriangle(rot(tip), rot(baseL), rot(baseR), theme.HUDMapMarkerBack)
	cv.FillTriangle(rot(tip), rot(noseL), rot(noseR), theme.HUDMapMarkerFront)
}

// DrawMapLegend は記号・色・種別名の対応を地図の下へ並べて描く。色見本に格子と同じ記号を重ね、
// 地図上の1文字から凡例を引けるようにする。全画面図の下部 chrome。top は並べ始める y ピクセル。
func DrawMapLegend(cv uicore.Canvas, face, glyphFace text.Face, top int) {
	const swatch = 14
	x, y := 8, top
	for _, g := range overworld.LegendGlyphs() {
		cv.FillRect(image.Rect(x, y, x+swatch, y+swatch), macroGlyphColor(g.Label), uicore.RectOptions{})
		drawCenteredGlyph(cv, string(g.Label), glyphFace, x, y, swatch, theme.OverworldMapGlyphText)
		cv.DrawText(image.Pt(x+20, y-2), g.Name, face, theme.TextPrimary)
		// 1項目120px幅で並べ、モーダル幅に収まる右端720pxを超えたら次の行へ折り返す
		x += 120
		if x > 720 {
			x, y = 8, y+22
		}
	}
	cv.DrawText(image.Pt(8, y+26), "N / Esc to close", face, theme.TextPrimary)
}

// drawMapRoad はチャンクセルを通る道を接続方角ごとに、セル中央から辺の中点へ細い矩形で引く。
// 太さはセル辺の1/5で最低1px。縮小地図でも街道の走りが読める。
func drawMapRoad(cv uicore.Canvas, x, y, cell int, road overworld.RoadDir) {
	t := max(cell/5, 1)
	half := t / 2
	ccx, ccy := x+cell/2, y+cell/2
	// 水平の道は横帯 bandTop..bandTop+t、垂直の道は縦帯 bandLeft..bandLeft+t を共有する。西と東は
	// 中央で重なり、image.Rect の Max 排他で t が奇数でも隙間なく接する。南北も同じ
	bandTop := ccy - half
	bandLeft := ccx - half
	col := theme.OverworldMapRoad
	if road&overworld.RoadW != 0 {
		cv.FillRect(image.Rect(x, bandTop, ccx+half, bandTop+t), col, uicore.RectOptions{})
	}
	if road&overworld.RoadE != 0 {
		cv.FillRect(image.Rect(bandLeft, bandTop, x+cell, bandTop+t), col, uicore.RectOptions{})
	}
	if road&overworld.RoadN != 0 {
		cv.FillRect(image.Rect(bandLeft, y, bandLeft+t, ccy+half), col, uicore.RectOptions{})
	}
	if road&overworld.RoadS != 0 {
		cv.FillRect(image.Rect(bandLeft, bandTop, bandLeft+t, y+cell), col, uicore.RectOptions{})
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
