package menuframe

import (
	"image"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/internal/uicore"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// noteRowH は補助フェイスを載せる1行の高さ。操作ヒント・説明・版の注記・入力欄のエラーに使う。
// 役割が同じ行はどの画面でも同じ高さにして、画面をまたいだ見た目のずれを防ぐ
const noteRowH = 16

// panelBackground はパネル背景のテクスチャを敷く。
func panelBackground(c *uicore.Container, res resources.UIResources) *uicore.Container {
	return c.SetBackgroundNineSlice(res.PanelBG.Image, res.PanelBG.BX, res.PanelBG.BY)
}

// PanelBox はパネルテクスチャを敷いた縦積みの箱を返す。行高・余白・背景は標準の既定に従う。
// TabScreen・PanelScreen に収まらない独自配置の画面が、意匠だけを部品へ合わせるのに使う。
// 置き場所は呼び出し側が Layout で決める。
func PanelBox(res resources.UIResources, content ...uicore.Drawable) uicore.Widget {
	return panelBackground(uicore.Panel(uicore.BoxStyle{}, theme.MenuPanelRowH, uicore.Placeable(content)...), res).SetPadding(theme.MenuPad)
}

// footerRow はフッタ行を組む。左にヘルプ、右端にページ表示を並べる。
// ヘルプ列を0幅で伸ばして余りを埋め、ページ表示を右へ寄せる。末尾に余白列を置いて右端を空ける。
// pager が空ならページ表示は出ない。
func footerRow(footer, pager string, res resources.UIResources) *uicore.Container {
	face := res.Text.SmallFace
	help := uicore.NewText(footer, face, theme.TextSecondary)
	help.VCenter = true
	pg := uicore.NewText(pager, face, theme.TextPrimary)
	pg.Align = uicore.AlignRight
	pg.VCenter = true
	gap := uicore.NewText("", face, theme.TextSecondary)
	return uicore.Row([]int{0, theme.MenuPagerW, theme.Space3}, help, pg, gap)
}

// PanelScreen は見出し・内容行・フッタを1枚のパネルへ縦に並べ、上端を固定して配置して返す。
// 背景はパネルテクスチャを敷く。内容は RenderList が返す行をそのまま渡す。ページ表示は
// フッタ行の右端に置く。上端固定により、項目数が違ってもタイトル・先頭項目が常に同じ位置に来る。
// 選択メニュー choice や設定メニューなど、項目数相応に伸びるパネル画面で使う。
func PanelScreen(world w.World, res resources.UIResources, title string, content []uicore.Drawable, footer, pager string) uicore.Widget {
	return panelScreen(world, res, title, content, footer, pager, theme.MenuPanelRowH)
}

// PanelScreenDense は PanelScreen の密行版。行間を空けるコマンドメニューと違い、
// キー一覧のような表を詰まった行高で項目数相応の小さなパネルに収める。
// 項目が多くてもログ領域に被らずに収まる。
func PanelScreenDense(world w.World, res resources.UIResources, title string, content []uicore.Drawable, footer, pager string) uicore.Widget {
	return panelScreen(world, res, title, content, footer, pager, theme.MenuTabRowH)
}

// panelScreen はパネル画面の実体。行高だけを密度の変種として受け取る。
func panelScreen(world w.World, res resources.UIResources, title string, content []uicore.Drawable, footer, pager string, rowH int) uicore.Widget {
	face := res.Text.BodyFace
	items := make([]uicore.FlexItem, 0, len(content)+3)
	if title != "" {
		items = append(items, uicore.FlexItem{W: uicore.NewText(title, face, theme.TextPrimary), Height: rowH})
	}
	for _, c := range uicore.Placeable(content) {
		items = append(items, uicore.FlexItem{W: c, Height: rowH})
	}
	if footer != "" || pager != "" {
		// フッタは内容から一行空けて置く
		items = append(items, uicore.FlexItem{Height: rowH})
		items = append(items, uicore.FlexItem{W: footerRow(footer, pager, res), Height: rowH})
	}

	// パネルの高さは内容に合わせる。横は画面中央、縦は上端を固定する。項目数が違っても
	// パネルの開始位置がそろい、メニュー間でタイトル・先頭項目の位置がずれない
	panelW := theme.MenuRowWidth + theme.MenuPad*2
	panelH := len(items)*rowH + theme.MenuPad*2
	crect := WindowRect(world)
	x := crect.Min.X + (crect.Dx()-panelW)/2
	y := crect.Min.Y
	rect := image.Rect(x, y, x+panelW, y+panelH)
	inner := image.Rect(rect.Min.X+theme.MenuPad, rect.Min.Y+theme.MenuPad, rect.Max.X-theme.MenuPad, rect.Max.Y-theme.MenuPad)
	uicore.FlexColumn(inner, items)

	return groupWithPanelBG(rect, res, items)
}

// tabBar はタブ帯を組む。ラベルを等幅で横に並べ、選択中のタブは金色の選択バーを敷き主色で、
// 他は補助色で描く。
func tabBar(labels []string, selected int, totalWidth int, face text.Face, selBar *resources.NineSliceTex) *uicore.Container {
	n := len(labels)
	widths := make([]int, n)
	cells := make([]uicore.Widget, n)
	for i, label := range labels {
		widths[i] = totalWidth / n
		clr := theme.TextSecondary
		if i == selected {
			clr = theme.TextSelected
		}
		t := uicore.NewText(label, face, clr)
		t.Align = uicore.AlignCenter
		t.VCenter = true
		if i == selected {
			// 選択タブは金色の選択バーを敷いたセルにする
			cells[i] = tabSelCell(t, selBar)
		} else {
			cells[i] = t
		}
	}
	return uicore.Row(widths, cells...)
}

// tabSelCell は子を金色の選択バーの上に置くセルを返す。選択タブの強調に使う。
func tabSelCell(child uicore.Widget, selBar *resources.NineSliceTex) *uicore.Container {
	cell := uicore.VBox(theme.MenuTabRowH, child)
	if selBar != nil {
		cell.SetBackgroundNineSlice(selBar.Image, selBar.BX, selBar.BY)
	}
	return cell
}

// TabScreen は見出し・タブ帯・内容行・フッタを1枚のモーダルに縦へ並べて返す。
// モーダルは中央固定枠 ModalRect いっぱいに広げる。タブ帯と一覧の間に一行の余白を空ける。
// フッタは flex-grow のスペーサで常に下端へ固定し、内容の量によらずヘルプの位置がぶれない。
// 容量の逆算や空行埋めは要らない。ページ表示はフッタ行の右端に置く。
func TabScreen(world w.World, res resources.UIResources, header string, tabLabels []string, tabIndex int, content []uicore.Drawable, footer, pager string) uicore.Widget {
	face := res.Text.BodyFace
	rect := ModalRect(world)
	inner := image.Rect(rect.Min.X+theme.MenuPad, rect.Min.Y+theme.MenuPad, rect.Max.X-theme.MenuPad, rect.Max.Y-theme.MenuPad)

	var items []uicore.FlexItem
	if header != "" {
		h := uicore.NewText(header, face, theme.TextPrimary)
		h.Align = uicore.AlignCenter
		items = append(items, uicore.FlexItem{W: h, Height: theme.MenuTabRowH})
	}
	if len(tabLabels) > 0 {
		items = append(items, uicore.FlexItem{W: tabBar(tabLabels, tabIndex, inner.Dx(), face, res.SelectionBar), Height: theme.MenuTabRowH})
	}
	// タブ帯と一覧を離す。詰まりすぎないよう一行空ける
	if header != "" || len(tabLabels) > 0 {
		items = append(items, uicore.FlexItem{Height: theme.MenuTabRowH})
	}
	for _, c := range uicore.Placeable(content) {
		items = append(items, uicore.FlexItem{W: c, Height: theme.MenuTabRowH})
	}
	if footer != "" || pager != "" {
		items = append(items, uicore.FlexItem{Grow: true})
		items = append(items, uicore.FlexItem{W: footerRow(footer, pager, res), Height: theme.MenuTabRowH})
	}
	uicore.FlexColumn(inner, items)

	return groupWithPanelBG(rect, res, items)
}

// groupWithPanelBG はパネルテクスチャの背景と、配置済みの flex 行を1つの Group に束ねる。
func groupWithPanelBG(rect image.Rectangle, res resources.UIResources, items []uicore.FlexItem) uicore.Widget {
	bg := uicore.NewNineSlice(res.PanelBG.Image, res.PanelBG.BX, res.PanelBG.BY)
	bg.Layout(rect)
	children := make([]uicore.Widget, 0, len(items)+1)
	children = append(children, bg)
	for _, it := range items {
		if it.W != nil {
			children = append(children, it.W)
		}
	}
	group := uicore.NewGroup(children...)
	group.Layout(rect)
	return group
}
