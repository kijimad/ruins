package menuframe

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/vrt"
	ui "github.com/kijimaD/ruins/internal/widgets/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// 部品カタログ。Storybook のように部品単体を固定入力で実描画し、見た目を部品の粒度で
// golden に固定する。画面全体の golden より、崩れたとき原因の部品を直接特定できる。
// 分子(フッタ行・タブ帯・一覧行)から有機体(入力欄・パネル・画面枠)までを覆う。

// storyIcon はカタログ用の決定的なアイコン。スプライト読み込みに依存せず単色で作る。
func storyIcon() *ebiten.Image {
	img := ebiten.NewImage(16, 16)
	img.Fill(color.RGBA{R: 200, G: 170, B: 60, A: 255})
	return img
}

// storyRows はカタログ用の一覧。見出し・選択中・非選択・アイコン・右寄せ数値を1組に収める。
func storyRows(res resources.UIResources) []ui.Widget {
	cols := styled.Cols(styled.Icon(), styled.Name(200), styled.Num(60))
	rows := []Row{
		{Cells: styled.TextCells("Weapons", "", ""), Header: true},
		{Cells: []styled.Cell{styled.IconCell(storyIcon()), styled.TextCell("Long Sword"), styled.TextCell("1.20kg")}},
		{Cells: []styled.Cell{styled.IconCell(storyIcon()), styled.TextCell("Short Bow"), styled.TextCell("0.80kg")}},
	}
	items, _ := RenderList(1, rows, cols, ListOpts{}, res)
	return items
}

// drawStory は widget 群を縦に積んで実描画し、golden と比較する。
func drawStory(t *testing.T, name string, size image.Point, rowH int, ws []ui.Widget) {
	t.Helper()
	items := make([]ui.FlexItem, len(ws))
	for i, wgt := range ws {
		items[i] = ui.FlexItem{W: wgt, Height: rowH}
	}
	ui.FlexColumn(image.Rect(0, 0, size.X, size.Y), items)
	screen := ebiten.NewImage(size.X, size.Y)
	cv := ui.NewEbitenCanvas(screen)
	for _, wgt := range ws {
		wgt.Draw(cv)
	}
	vrt.AssertFrameGolden(t, name, screen)
}

// drawWidget は配置済みの widget を1枚だけ実描画し、golden と比較する。
func drawWidget(t *testing.T, name string, size image.Point, wgt ui.Widget) {
	t.Helper()
	screen := ebiten.NewImage(size.X, size.Y)
	wgt.Draw(ui.NewEbitenCanvas(screen))
	vrt.AssertFrameGolden(t, name, screen)
}

func TestGolden_Story_FooterRow(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	row := footerRow("? Help", "", res)
	row.Layout(image.Rect(0, 0, 360, theme.MenuTabRowH))
	drawWidget(t, "TestGolden_Story_FooterRow", image.Pt(360, theme.MenuTabRowH), row)
}

func TestGolden_Story_FooterRowWithPager(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	row := footerRow("? Help", "3/7", res)
	row.Layout(image.Rect(0, 0, 360, theme.MenuTabRowH))
	drawWidget(t, "TestGolden_Story_FooterRowWithPager", image.Pt(360, theme.MenuTabRowH), row)
}

func TestGolden_Story_TabBar(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	bar := tabBar([]string{"Equipment", "Skills", "Health"}, 1, 360, res.Text.BodyFace, res.SelectionBar)
	bar.Layout(image.Rect(0, 0, 360, theme.MenuTabRowH))
	drawWidget(t, "TestGolden_Story_TabBar", image.Pt(360, theme.MenuTabRowH), bar)
}

func TestGolden_Story_ListRows(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	drawStory(t, "TestGolden_Story_ListRows", image.Pt(300, theme.MenuTabRowH*3), theme.MenuTabRowH, storyRows(res))
}

func TestGolden_Story_InputBox(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	body := ui.NewText("Ash|", res.Text.BodyFace, theme.TextPrimary)
	body.VCenter = true
	box := InputBox(res, body)
	box.Layout(image.Rect(0, 0, 320, 44))
	drawWidget(t, "TestGolden_Story_InputBox", image.Pt(320, 44), box)
}

func TestGolden_Story_PanelBox(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	box := PanelBox(res,
		ui.NewText("Vitality 10", res.Text.BodyFace, theme.TextPrimary),
		ui.NewText("Strength 12", res.Text.BodyFace, theme.TextPrimary),
	)
	box.Layout(image.Rect(0, 0, 200, theme.MenuPanelRowH*2+theme.MenuPad*2))
	drawWidget(t, "TestGolden_Story_PanelBox", image.Pt(200, theme.MenuPanelRowH*2+theme.MenuPad*2), box)
}

func TestGolden_Story_PanelScreen(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	screenW, screenH := storyScreenSize(world)
	panel := PanelScreen(world, res, "Settings", storyRows(res), "? Help", "2/5")
	drawWidget(t, "TestGolden_Story_PanelScreen", image.Pt(screenW, screenH), panel)
}

func TestGolden_Story_PanelScreenDense(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	screenW, screenH := storyScreenSize(world)
	panel := PanelScreenDense(world, res, "Key bindings", storyRows(res), "", "")
	drawWidget(t, "TestGolden_Story_PanelScreenDense", image.Pt(screenW, screenH), panel)
}

func TestGolden_Story_TabScreen(t *testing.T) {
	t.Parallel()
	world := vrt.InitUIWorld(t)
	res := world.Resources.UIResources
	screenW, screenH := storyScreenSize(world)
	screen := TabScreen(world, res, "Ash", []string{"Equipment", "Skills", "Health"}, 1, storyRows(res), "? Help", "1/2")
	drawWidget(t, "TestGolden_Story_TabScreen", image.Pt(screenW, screenH), screen)
}

// storyScreenSize は画面枠の部品が配置される論理画面の大きさを返す。
func storyScreenSize(world w.World) (int, int) {
	return world.Resources.ScreenDimensions.Width, world.Resources.ScreenDimensions.Height
}
