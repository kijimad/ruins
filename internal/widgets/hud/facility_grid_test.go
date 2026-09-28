package hud

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDrawFacilityGrid_描画命令 は描画コンテキスト無しで、パネル・セル・カーソル・情報行の
// 描画命令が出ることを固定する。llvmpipe を要する golden と違い通常のテストで回りカバレッジに載る。
func TestDrawFacilityGrid_描画命令(t *testing.T) {
	t.Parallel()
	var cv fakeCanvas
	view := FacilityGridView{
		Cols: 2,
		Rows: 2,
		Cells: []FacilityCell{
			{Kind: FacilityCellCube}, {Kind: FacilityCellEmpty},
			{Kind: FacilityCellUsed}, {Kind: FacilityCellBlocked},
		},
		CursorCol: 1,
		CursorRow: 0,
		Title:     "Facility",
		Content:   "Empty",
		Hint:      "Enter: place",
		Footer:    "Esc: Close",
	}

	DrawFacilityGrid(&cv, image.Rect(0, 0, 400, 300), nil, view)

	// パネル背景1 + 4マスの下地。いずれも角丸塗り
	assert.Equal(t, 5, cv.roundedFills)
	// パネル枠1 + カーソル枠1
	assert.Equal(t, 2, cv.roundedStrokes)
	// タイトル・内容・フッタの3行
	assert.Len(t, cv.texts, 3)
}
