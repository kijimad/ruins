package hud_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

func TestGolden_Story_FacilityGrid(t *testing.T) {
	t.Parallel()
	res := storyRes(t)
	// キューブ本体と据えた設備のスプライトを模した無地アイコン。DrawImageRect の枝を通す
	icon := ebiten.NewImage(16, 16)
	icon.Fill(color.RGBA{R: 200, G: 200, B: 200, A: 255})
	view := hud.FacilityGridView{
		Cols: 3,
		Rows: 3,
		Cells: []hud.FacilityCell{
			{Kind: hud.FacilityCellUsed, Icon: icon}, {Kind: hud.FacilityCellEmpty}, {Kind: hud.FacilityCellEmpty},
			{Kind: hud.FacilityCellEmpty}, {Kind: hud.FacilityCellCube, Icon: icon}, {Kind: hud.FacilityCellEmpty},
			{Kind: hud.FacilityCellEmpty}, {Kind: hud.FacilityCellEmpty}, {Kind: hud.FacilityCellBlocked},
		},
		CursorCol: 0,
		CursorRow: 1,
		Title:     "Facility",
		Content:   "Empty",
		Hint:      "Enter: place",
		Footer:    "Arrows: Move  Esc: Close",
	}
	rect := image.Rect(0, 0, storyScreen.Width, storyScreen.Height)
	vrt.AssertScreenGolden(t, func() func(screen *ebiten.Image) {
		return func(screen *ebiten.Image) {
			hud.DrawFacilityGrid(uicore.NewEbitenCanvas(screen), rect, res.Text.BodyFace, view)
		}
	}, storyScreen.Width, storyScreen.Height)
}
