package hud_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// TestGolden_Story_RoadPlan は道の計画の途中経過を、俯瞰図と同じ描き方で段階ごとに並べる。開始の行を含む計画区間を、
// 区域と墜落現場、行き先の予約、経路を1本ずつ、集落、点在ランドマークの順に左上から描く
func TestGolden_Story_RoadPlan(t *testing.T) {
	t.Parallel()
	res := storyRes(t)
	raws, err := loader.LoadRaws()
	require.NoError(t, err)
	snaps := overworld.RoadPlanSnapshots(raws, 12345, 0, 9, 4)
	require.NotEmpty(t, snaps)

	const (
		cellPx  = 12
		gap     = 16
		labelPx = 18
		perRow  = 8
	)
	rows, cols := len(snaps[0].View.Cells), len(snaps[0].View.Cells[0])
	panelW, panelH := cols*cellPx, rows*cellPx
	lines := (len(snaps) + perRow - 1) / perRow
	width := gap + perRow*(panelW+gap)
	height := gap + lines*(labelPx+panelH+gap)
	vrt.AssertScreenGolden(t, func() func(screen *ebiten.Image) {
		return func(screen *ebiten.Image) {
			cv := uicore.NewEbitenCanvas(screen)
			cv.FillRect(image.Rect(0, 0, width, height), theme.OverworldMapUnknownGlyph, uicore.RectOptions{})
			for i, s := range snaps {
				x := gap + (i%perRow)*(panelW+gap)
				y := gap + (i/perRow)*(labelPx+panelH+gap)
				cv.DrawText(image.Point{X: x, Y: y}, s.Label, res.Text.SmallFace, theme.OverworldMapGlyphText)
				hud.DrawMapGrid(cv, s.View, hud.MapGridStyle{OriginX: x, OriginY: y + labelPx, CellPx: cellPx, GlyphFace: res.Text.SmallFace})
			}
		}
	}, width, height)
}
