package hud

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// FacilityCellKind は設備グリッドの1マスの見た目の種別。据付可否と下地色を分ける。
type FacilityCellKind int

// 設備グリッドのマス種別。据付可否と下地色を分ける
const (
	FacilityCellCube    FacilityCellKind = iota // キューブ本体
	FacilityCellEmpty                           // 空き。据付できる
	FacilityCellUsed                            // 設備が据わっている
	FacilityCellBlocked                         // 壁や障害物。据付できない
)

// FacilityCell は設備グリッドの1マス。下地の種別と、重ねるスプライトを持つ。
type FacilityCell struct {
	Kind FacilityCellKind
	Icon *ebiten.Image // キューブや設備のスプライト。無ければ nil
}

// FacilityGridView は設備画面の描画に要る値。展開空間の各マスとカーソル、選択マスの説明を持つ。
// ドメインの判定は呼び出し側が済ませ、ここは見た目だけを受け取る。
type FacilityGridView struct {
	Cols, Rows           int
	Cells                []FacilityCell // 行優先で Cols*Rows 個
	CursorCol, CursorRow int
	Title                string
	Content              string // カーソルのマスの内容名
	Hint                 string // カーソルのマスで押せる操作。無ければ空
	Footer               string // 共通の操作説明
}

// facilityGridCellPx は設備グリッドの1マスの辺
const facilityGridCellPx = 44

// DrawFacilityGrid は設備画面のパネル・グリッド・カーソル・説明を cv へ描く。画面側は view を組んで渡すだけで、
// uicore の組み立てはここへ閉じる。
func DrawFacilityGrid(cv uicore.Canvas, rect image.Rectangle, face text.Face, view FacilityGridView) {
	Chrome{}.Panel(cv, rect)
	cv.DrawText(image.Pt(rect.Min.X+16, rect.Min.Y+12), view.Title, face, theme.TextPrimary)

	gridW := view.Cols * facilityGridCellPx
	gridH := view.Rows * facilityGridCellPx
	originX := rect.Min.X + (rect.Dx()-gridW)/2
	originY := rect.Min.Y + (rect.Dy()-gridH)/2
	for row := range view.Rows {
		for col := range view.Cols {
			cell := view.Cells[row*view.Cols+col]
			r := image.Rect(originX+col*facilityGridCellPx, originY+row*facilityGridCellPx,
				originX+(col+1)*facilityGridCellPx, originY+(row+1)*facilityGridCellPx)
			cv.FillRect(r.Inset(2), facilityCellColor(cell.Kind), uicore.RectOptions{Radius: 3})
			if cell.Icon != nil {
				cv.DrawImageRect(r.Inset(6), cell.Icon)
			}
			if col == view.CursorCol && row == view.CursorRow {
				cv.StrokeRect(r.Inset(1), 2, theme.HUDSlotSelectedBorder, uicore.RectOptions{Radius: 4})
			}
		}
	}

	cv.DrawText(image.Pt(rect.Min.X+16, rect.Max.Y-52), view.Content, face, theme.TextPrimary)
	footer := view.Footer
	if view.Hint != "" {
		footer = view.Hint + "  " + footer
	}
	cv.DrawText(image.Pt(rect.Min.X+16, rect.Max.Y-28), footer, face, theme.TextDisabled)
}

// facilityCellColor はマス種別ごとの下地色を返す
func facilityCellColor(kind FacilityCellKind) color.Color {
	switch kind {
	case FacilityCellCube:
		return theme.PanelHighlight
	case FacilityCellUsed:
		return theme.StatusSuccess
	case FacilityCellBlocked:
		return theme.HUDGaugeBg
	default:
		return theme.ListSelectedBg
	}
}
