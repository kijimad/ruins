package menuframe

import (
	"image"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// 背景を透かすタイトル画面の寸法。画面はこれらを知らない
const (
	titleMenuLeft   = 64  // メニューの左端
	titleMenuBottom = 72  // メニュー下端から画面下端までの高さ
	titleNoteW      = 232 // 右下の注記の幅
	titleNoteMargin = 8   // 注記と画面端の余白
)

// TitleScreen は背景を透かすタイトル画面を組む。メニューを左下へ左寄せで置き、
// 版などの注記を右下へ右寄せで積む。パネル背景は付けない。
// 寸法と余白は部品が持ち、画面は項目と注記の文言だけを渡す。
func TitleScreen(world w.World, res resources.UIResources, itemIndex int, rows []Row, notes []string) ui.Widget {
	sd := world.Resources.ScreenDimensions

	// タイトル画面は単一ページ。ページ表示は使わないので捨てる
	listRows, _ := RenderList(itemIndex, rows, styled.Cols(styled.Name()), ListOpts{}, res)
	menu := ui.VBox(theme.MenuPanelRowH, ui.Placeable(listRows)...)
	menuH := len(listRows) * theme.MenuPanelRowH
	menuTop := sd.Height - titleMenuBottom - menuH
	menu.Layout(image.Rect(titleMenuLeft, menuTop, titleMenuLeft+theme.MenuRowWidth, menuTop+menuH))

	children := make([]ui.Widget, 0, 1+len(notes))
	children = append(children, menu)
	// 注記は下端から上へ積む。行数が変わっても最終行の位置は動かない
	for i, line := range notes {
		t := ui.NewText(line, res.Text.SmallFace, theme.TextAccent)
		t.Align = ui.AlignRight
		y := sd.Height - noteRowH*(len(notes)-i) - titleNoteMargin
		t.Layout(image.Rect(sd.Width-titleNoteW-titleNoteMargin, y, sd.Width-titleNoteMargin, y+noteRowH))
		children = append(children, t)
	}

	root := ui.NewGroup(children...)
	root.Layout(image.Rect(0, 0, sd.Width, sd.Height))
	return root
}
