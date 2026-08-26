package states

import (
	"image"
	"image/color"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

const (
	// panelScreenRowH は簡易メニューの行高。コマンドメニュー向けに行間を空ける。
	panelScreenRowH = 26
	// panelScreenPad はパネルの内側余白。
	panelScreenPad = 12
)

// panelBackground はパネル背景のテクスチャを敷く。ebitenui 時代の res.Panel.Image と同じ意匠。
func panelBackground(c *ui.Container, res resources.UIResources) *ui.Container {
	return c.SetBackgroundNineSlice(res.PanelBG.Image, res.PanelBG.BX, res.PanelBG.BY)
}

// buildPanelScreenUI は見出し・内容行・フッタを1枚のパネルへ縦に並べ、画面中央に配置して返す。
// 項目数相応の小さなパネル。背景はパネルテクスチャを敷く。内容は renderMenuListUI が返す行をそのまま渡す。
func buildPanelScreenUI(world w.World, res resources.UIResources, title string, content []ui.Widget, footer string) ui.Widget {
	face := res.Text.BodyFace
	items := make([]ui.Widget, 0, len(content)+2)
	if title != "" {
		items = append(items, ui.NewText(title, face, theme.TextPrimary))
	}
	items = append(items, content...)
	if footer != "" {
		items = append(items, ui.NewText(footer, face, theme.TextSecondary))
	}

	panelW := menuRowWidth + panelScreenPad*2
	panelH := len(items)*panelScreenRowH + panelScreenPad*2
	panel := panelBackground(ui.Panel(ui.BoxStyle{}, panelScreenRowH, items...), res).SetPadding(panelScreenPad)

	rect := menuframe.CenterWindowRect(world)
	x := rect.Min.X + (rect.Dx()-panelW)/2
	y := max(rect.Min.Y+(rect.Dy()-panelH)/2, rect.Min.Y)
	panel.Layout(image.Rect(x, y, x+panelW, y+panelH))
	return panel
}

// tabBarUI はタブ帯を組む。ラベルを等幅で横に並べ、選択中のタブは金色の選択バーを敷き主色で、
// 他は補助色で描く。ebitenui 時代の NewTabBar と同じ意匠。
func tabBarUI(labels []string, selected int, totalWidth int, face text.Face, selBar *resources.NineSliceTex) *ui.Container {
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
			cells[i] = panelSelCell(t, selBar)
		} else {
			cells[i] = t
		}
	}
	return ui.Row(widths, cells...)
}

// panelSelCell は子を金色の選択バーの上に置くセルを返す。選択タブの強調に使う。
func panelSelCell(child ui.Widget, selBar *resources.NineSliceTex) *ui.Container {
	cell := ui.VBox(panelScreenRowH, child)
	if selBar != nil {
		cell.SetBackgroundNineSlice(selBar.Image, selBar.BX, selBar.BY)
	}
	return cell
}

// buildTabScreenUI は見出し・タブ帯・内容行・フッタを1枚のモーダルに縦へ並べて返す。
// NewTabScreen の internal/ui 版。モーダルは中央固定枠いっぱいに広げ、フッタは空行で下端へ寄せる。
func buildTabScreenUI(world w.World, res resources.UIResources, header string, tabLabels []string, tabIndex int, content []ui.Widget, footer string) ui.Widget {
	face := res.Text.BodyFace
	rect := menuframe.ModalRect(world)
	innerW := rect.Dx() - panelScreenPad*2

	var items []ui.Widget
	if header != "" {
		h := ui.NewText(header, face, theme.TextPrimary)
		h.Align = ui.AlignCenter
		items = append(items, ui.Row([]int{innerW}, h))
	}
	if len(tabLabels) > 0 {
		items = append(items, tabBarUI(tabLabels, tabIndex, innerW, face, res.SelectionBar))
	}
	items = append(items, content...)

	// フッタは下端へ寄せる。内容の下を空行で埋め、最下段にフッタを置く
	if footer != "" {
		capacity := (rect.Dy() - panelScreenPad*2) / panelScreenRowH
		for len(items) < capacity-1 {
			items = append(items, ui.NewText(" ", face, theme.TextPrimary))
		}
		items = append(items, ui.NewText(footer, face, theme.TextSecondary))
	}

	panel := panelBackground(ui.Panel(ui.BoxStyle{}, panelScreenRowH, items...), res).SetPadding(panelScreenPad)
	panel.Layout(rect)
	return panel
}

// toUIAlign は styled のそろえを internal/ui のそろえへ写す。
func toUIAlign(a styled.TextAlign) ui.Align {
	if a == styled.AlignRight {
		return ui.AlignRight
	}
	return ui.AlignLeft
}

// sumWidths は列幅の合計を返す。ページ表示行を全幅で置くのに使う。
func sumWidths(colWidths []int) int {
	total := 0
	for _, wdt := range colWidths {
		total += wdt
	}
	return total
}

// renderMenuListUI は renderMenuList の internal/ui 版。行データからページ送り・ページ表示・
// 見出し・選択強調・空行埋めを備えた行ウィジェット列を返す。呼び出し側はこれをパネルに並べる。
// 1ページの件数は呼び出し側が opts.ItemsPerPage に解決して渡す。0 なら全行を1ページに収める。
func renderMenuListUI(itemIndex int, rows []menuRow, colWidths []int, aligns []styled.TextAlign, opts menuListOpts, res resources.UIResources) []ui.Widget {
	face := res.Text.BodyFace
	perPage := opts.ItemsPerPage
	if perPage <= 0 {
		perPage = max(len(rows), 1)
	}
	pg := pagination.New(itemIndex, len(rows), perPage)

	var items []ui.Widget
	if opts.AlwaysIndicator || pg.IsEnabled() {
		pageText := pg.GetPageText()
		if pageText == "" {
			pageText = " "
		}
		ind := ui.NewText(pageText, face, theme.TextSecondary)
		ind.Align = ui.AlignCenter
		ind.VCenter = true
		items = append(items, ui.Row([]int{sumWidths(colWidths)}, ind))
	}
	if opts.HeaderRow != nil {
		items = append(items, headerRowUI(opts.HeaderRow, colWidths, face))
	}
	visible := pagination.VisibleEntries(rows, pg)
	for _, entry := range visible {
		if entry.Item.Header {
			items = append(items, headerRowUI(cellTexts(entry.Item.Cells), colWidths, face))
			continue
		}
		items = append(items, dataRowUI(entry.Item.Cells, colWidths, aligns, pg.IsSelectedInPage(entry.Index), face, res))
	}
	// 複数ページの画面は各ページを1ページ件数ぶんの空行で埋め、ページを繰っても高さを一定にする
	if len(rows) > perPage {
		for i := len(visible); i < perPage; i++ {
			items = append(items, blankRowUI(colWidths, face))
		}
	}
	// 行が無いときの空表示を一覧側で持つ。各メニューが同じ後処理を書かずに済む
	if len(rows) == 0 && opts.EmptyText != "" {
		items = append(items, ui.NewText(opts.EmptyText, face, theme.TextSecondary))
	}
	return items
}

// headerRowUI は見出し行を組む。カーソルは止まらず、補助色で描く。
func headerRowUI(texts []string, colWidths []int, face text.Face) *ui.Container {
	cells := make([]ui.Widget, len(texts))
	for i, s := range texts {
		t := ui.NewText(s, face, theme.TextSecondary)
		t.VCenter = true
		cells[i] = t
	}
	return ui.Row(colWidths, cells...)
}

// dataRowUI はデータ行を組む。選択中なら金色の選択バーを敷き文字色を選択色にする。アイコンセルは画像で描く。
func dataRowUI(cells []styled.Cell, colWidths []int, aligns []styled.TextAlign, selected bool, face text.Face, res resources.UIResources) *ui.Container {
	selBar := res.SelectionBar
	// 非選択は暗く、選択は明るく。元の NewTableRow と同じ色分けでカーソル位置を際立たせる
	var textColor color.Color = theme.TextSecondary
	if selected {
		textColor = theme.TextSelected
	}
	cellWidgets := make([]ui.Widget, len(cells))
	for i, c := range cells {
		if c.Icon != nil {
			cellWidgets[i] = ui.NewGraphic(c.Icon)
			continue
		}
		t := ui.NewText(c.Text, face, textColor)
		t.VCenter = true
		if i < len(aligns) {
			t.Align = toUIAlign(aligns[i])
		}
		cellWidgets[i] = t
	}
	row := ui.Row(colWidths, cellWidgets...)
	if selected && selBar != nil {
		row.SetBackgroundNineSlice(selBar.Image, selBar.BX, selBar.BY)
	}
	// 行の下にグラデーションの区切り線を敷く。元の NewTableRow と同じ意匠。
	// RowDivider は非乗算済みの値なので NRGBA として色を掛ける
	if res.GradientLine != nil {
		row.SetBottomLine(res.GradientLine, color.NRGBA(theme.RowDivider))
	}
	return row
}

// blankRowUI は高さを揃えるための空行を組む。
func blankRowUI(colWidths []int, face text.Face) *ui.Container {
	cells := make([]ui.Widget, len(colWidths))
	for i := range cells {
		cells[i] = ui.NewText(" ", face, theme.TextPrimary)
	}
	return ui.Row(colWidths, cells...)
}
