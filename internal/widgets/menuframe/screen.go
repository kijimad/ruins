package menuframe

import (
	"image"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// panelBackground はパネル背景のテクスチャを敷く。ebitenui 時代の res.Panel.Image と同じ意匠。
func panelBackground(c *ui.Container, res resources.UIResources) *ui.Container {
	return c.SetBackgroundNineSlice(res.PanelBG.Image, res.PanelBG.BX, res.PanelBG.BY)
}

// PanelBox はパネルテクスチャを敷いた縦積みの箱を返す。行高・余白・背景は標準の既定に従う。
// TabScreen・PanelScreen に収まらない独自配置の画面が、意匠だけを部品へ合わせるのに使う。
// 置き場所は呼び出し側が Layout で決める。
func PanelBox(res resources.UIResources, content ...ui.Widget) *ui.Container {
	return panelBackground(ui.Panel(ui.BoxStyle{}, theme.MenuPanelRowH, content...), res).SetPadding(theme.MenuPad)
}

// footerRow はフッタ行を組む。左にヘルプ、右端にページ表示を並べる。
// ヘルプ列を0幅で伸ばして余りを埋め、ページ表示を右へ寄せる。末尾に余白列を置いて右端を空ける。
// pager が空ならページ表示は出ない。
func footerRow(footer, pager string, res resources.UIResources) *ui.Container {
	face := res.Text.SmallFace
	help := ui.NewText(footer, face, theme.TextSecondary)
	help.VCenter = true
	pg := ui.NewText(pager, face, theme.TextPrimary)
	pg.Align = ui.AlignRight
	pg.VCenter = true
	gap := ui.NewText("", face, theme.TextSecondary)
	return ui.Row([]int{0, theme.MenuPagerW, theme.Space3}, help, pg, gap)
}

// PanelScreen は見出し・内容行・フッタを1枚のパネルへ縦に並べ、上端を固定して配置して返す。
// 背景はパネルテクスチャを敷く。内容は RenderList が返す行をそのまま渡す。ページ表示は
// フッタ行の右端に置く。上端固定により、項目数が違ってもタイトル・先頭項目が常に同じ位置に来る。
// 選択メニュー choice や設定メニューなど、項目数相応に伸びるパネル画面で使う。
func PanelScreen(world w.World, res resources.UIResources, title string, content []ui.Widget, footer, pager string) ui.Widget {
	face := res.Text.BodyFace
	items := make([]ui.Widget, 0, len(content)+3)
	if title != "" {
		items = append(items, ui.NewText(title, face, theme.TextPrimary))
	}
	items = append(items, content...)
	if footer != "" || pager != "" {
		// フッタは内容から一行空けて置く。元のパネルと同じく内容と離す。
		// 空行は文字を持たせない。高さは行高が確保する
		items = append(items, ui.NewText("", face, theme.TextPrimary))
		items = append(items, footerRow(footer, pager, res))
	}

	panelW := theme.MenuRowWidth + theme.MenuPad*2
	panelH := len(items)*theme.MenuPanelRowH + theme.MenuPad*2
	panel := panelBackground(ui.Panel(ui.BoxStyle{}, theme.MenuPanelRowH, items...), res).SetPadding(theme.MenuPad)

	// 横は画面中央、縦は上端を固定する。項目数が違ってもパネルの開始位置がそろい、メニュー間で
	// タイトル・ページ表示・先頭項目の位置がずれない。全パネルを同じ規則で置く。
	crect := CenterWindowRect(world)
	x := crect.Min.X + (crect.Dx()-panelW)/2
	y := crect.Min.Y
	panel.Layout(image.Rect(x, y, x+panelW, y+panelH))
	return panel
}

// tabBar はタブ帯を組む。ラベルを等幅で横に並べ、選択中のタブは金色の選択バーを敷き主色で、
// 他は補助色で描く。ebitenui 時代の NewTabBar と同じ意匠。
func tabBar(labels []string, selected int, totalWidth int, face text.Face, selBar *resources.NineSliceTex) *ui.Container {
	n := len(labels)
	widths := make([]int, n)
	cells := make([]ui.Widget, n)
	for i, label := range labels {
		widths[i] = totalWidth / n
		clr := theme.TextSecondary
		if i == selected {
			clr = theme.TextSelected
		}
		t := ui.NewText(label, face, clr)
		t.Align = ui.AlignCenter
		t.VCenter = true
		if i == selected {
			// 選択タブは金色の選択バーを敷いたセルにする
			cells[i] = tabSelCell(t, selBar)
		} else {
			cells[i] = t
		}
	}
	return ui.Row(widths, cells...)
}

// tabSelCell は子を金色の選択バーの上に置くセルを返す。選択タブの強調に使う。
func tabSelCell(child ui.Widget, selBar *resources.NineSliceTex) *ui.Container {
	cell := ui.VBox(theme.MenuTabRowH, child)
	if selBar != nil {
		cell.SetBackgroundNineSlice(selBar.Image, selBar.BX, selBar.BY)
	}
	return cell
}

// TabScreen は見出し・タブ帯・内容行・フッタを1枚のモーダルに縦へ並べて返す。
// モーダルは中央固定枠 ModalRect いっぱいに広げる。タブ帯と一覧の間に一行の余白を空ける。
// フッタは常に下端へ固定し、内容の量によらずヘルプの位置がぶれないようにする。空行埋めの容量は
// ListCapacity と同じ ModalRect・行高・余白から算出するため、両者は同じパッケージで一貫する。
// ページ表示はフッタ行の右端に置く。
func TabScreen(world w.World, res resources.UIResources, header string, tabLabels []string, tabIndex int, content []ui.Widget, footer, pager string) ui.Widget {
	face := res.Text.BodyFace
	rect := ModalRect(world)
	innerW := rect.Dx() - theme.MenuPad*2

	var items []ui.Widget
	if header != "" {
		h := ui.NewText(header, face, theme.TextPrimary)
		h.Align = ui.AlignCenter
		items = append(items, ui.Row([]int{innerW}, h))
	}
	if len(tabLabels) > 0 {
		items = append(items, tabBar(tabLabels, tabIndex, innerW, face, res.SelectionBar))
	}
	// タブ帯と一覧を離す。詰まりすぎないよう一行空ける
	if header != "" || len(tabLabels) > 0 {
		items = append(items, ui.NewText("", face, theme.TextPrimary))
	}
	items = append(items, content...)

	// フッタは下端へ寄せる。内容の下を空行で埋め、最下段にフッタ行を置く。ヘルプの位置を固定する
	if footer != "" || pager != "" {
		capacity := (rect.Dy() - theme.MenuPad*2) / theme.MenuTabRowH
		for len(items) < capacity-1 {
			items = append(items, ui.NewText("", face, theme.TextPrimary))
		}
		items = append(items, footerRow(footer, pager, res))
	}

	panel := panelBackground(ui.Panel(ui.BoxStyle{}, theme.MenuTabRowH, items...), res).SetPadding(theme.MenuPad)
	panel.Layout(rect)
	return panel
}
