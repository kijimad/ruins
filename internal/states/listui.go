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
	// tabScreenRowH はタブ画面の一覧行の高さ。詰まった表として元の ebitenui の行密度に合わせる。
	// menuframe.capacityRowH と同じ値。ページ件数の計算と描画の行高を揃える。
	tabScreenRowH = 21
	// panelScreenRowH はパネル画面の行の高さ。コマンドメニュー向けに行間を空ける。
	// 元の renderMenuList の Spaced に相当する。
	panelScreenRowH = 30
	// panelScreenPad はパネルの内側余白。
	panelScreenPad = 12
	// footerPagerWidth はフッタ右端のページ表示列の幅。数字を右寄せで収める。
	footerPagerWidth = 60
)

// panelBackground はパネル背景のテクスチャを敷く。ebitenui 時代の res.Panel.Image と同じ意匠。
func panelBackground(c *ui.Container, res resources.UIResources) *ui.Container {
	return c.SetBackgroundNineSlice(res.PanelBG.Image, res.PanelBG.BX, res.PanelBG.BY)
}

// footerRowUI はフッタ行を組む。左にヘルプ、右端にページ表示を並べる。
// ヘルプ列を0幅で伸ばして余りを埋め、ページ表示を右へ寄せる。末尾に余白列を置いて右端を空ける。
// pager が空ならページ表示は出ない。
func footerRowUI(footer, pager string, res resources.UIResources) *ui.Container {
	face := res.Text.SmallFace
	help := ui.NewText(footer, face, theme.TextSecondary)
	help.VCenter = true
	pg := ui.NewText(pager, face, theme.TextPrimary)
	pg.Align = ui.AlignRight
	pg.VCenter = true
	gap := ui.NewText("", face, theme.TextSecondary)
	return ui.Row([]int{0, footerPagerWidth, theme.Space3}, help, pg, gap)
}

// buildPanelScreenUI は見出し・内容行・フッタを1枚のパネルへ縦に並べ、上端を固定して配置して返す。
// 背景はパネルテクスチャを敷く。内容は renderMenuListUI が返す行をそのまま渡す。ページ表示は
// フッタ行の右端に置く。上端固定により、項目数が違ってもタイトル・先頭項目が常に同じ位置に来る。
func buildPanelScreenUI(world w.World, res resources.UIResources, title string, content []ui.Widget, footer, pager string) ui.Widget {
	face := res.Text.BodyFace
	items := make([]ui.Widget, 0, len(content)+3)
	if title != "" {
		items = append(items, ui.NewText(title, face, theme.TextPrimary))
	}
	items = append(items, content...)
	if footer != "" || pager != "" {
		// フッタは内容から一行空けて置く。元のパネルと同じく内容と離す。
		items = append(items, ui.NewText(" ", face, theme.TextPrimary))
		items = append(items, footerRowUI(footer, pager, res))
	}

	panelW := menuRowWidth + panelScreenPad*2
	panelH := len(items)*panelScreenRowH + panelScreenPad*2
	panel := panelBackground(ui.Panel(ui.BoxStyle{}, panelScreenRowH, items...), res).SetPadding(panelScreenPad)

	// 横は画面中央、縦は上端を固定する。項目数が違ってもパネルの開始位置がそろい、メニュー間で
	// タイトル・ページ表示・先頭項目の位置がずれない。全パネルを同じ規則で置く。
	crect := menuframe.CenterWindowRect(world)
	x := crect.Min.X + (crect.Dx()-panelW)/2
	y := crect.Min.Y
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
	cell := ui.VBox(tabScreenRowH, child)
	if selBar != nil {
		cell.SetBackgroundNineSlice(selBar.Image, selBar.BX, selBar.BY)
	}
	return cell
}

// buildTabScreenUI は見出し・タブ帯・内容行・フッタを1枚のモーダルに縦へ並べて返す。
// NewTabScreen の internal/ui 版。モーダルは中央固定枠いっぱいに広げる。タブ帯と一覧の間、
// および一覧とフッタの間に一行の余白を空ける。フッタは一覧の直下に置き、下端へは押し込まない。
// ページ表示はフッタ行の右端に置く。
func buildTabScreenUI(world w.World, res resources.UIResources, header string, tabLabels []string, tabIndex int, content []ui.Widget, footer, pager string) ui.Widget {
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
	// タブ帯と一覧を離す。詰まりすぎないよう一行空ける
	if header != "" || len(tabLabels) > 0 {
		items = append(items, ui.NewText(" ", face, theme.TextPrimary))
	}
	items = append(items, content...)

	// フッタは一覧の一行下に置く。下端へ押し込まず、最後のエントリの近くに置く
	if footer != "" || pager != "" {
		items = append(items, ui.NewText(" ", face, theme.TextPrimary))
		items = append(items, footerRowUI(footer, pager, res))
	}

	panel := panelBackground(ui.Panel(ui.BoxStyle{}, tabScreenRowH, items...), res).SetPadding(panelScreenPad)
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

// renderMenuListUI は renderMenuList の internal/ui 版。行データから見出し・選択強調・空行埋めを
// 備えた行ウィジェット列と、フッタ右端へ出すページ表示文字列を返す。ページ表示は複数ページのとき
// だけ非空になる。呼び出し側は行をパネルに並べ、ページ表示を画面ビルダのフッタへ渡す。
// 1ページの件数は呼び出し側が opts.ItemsPerPage に解決して渡す。0 なら全行を1ページに収める。
func renderMenuListUI(itemIndex int, rows []menuRow, colWidths []int, aligns []styled.TextAlign, opts menuListOpts, res resources.UIResources) ([]ui.Widget, string) {
	face := res.Text.BodyFace
	perPage := opts.ItemsPerPage
	if perPage <= 0 {
		perPage = max(len(rows), 1)
	}
	pg := pagination.New(itemIndex, len(rows), perPage)

	var items []ui.Widget
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
	return items, pg.GetPageText()
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
