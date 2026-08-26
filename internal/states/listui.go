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

// panelScreenStyle は簡易メニューのパネルの背景と枠。
var panelScreenStyle = ui.BoxStyle{Fill: theme.WindowBackground, Border: theme.PanelHighlight, BorderWidth: 1}

// buildPanelScreenUI は見出し・内容行・フッタを1枚のパネルへ縦に並べ、画面中央に配置して返す。
// NewPanelScreen の internal/ui 版。内容は renderMenuListUI が返す行をそのまま渡す。
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
	panel := ui.Panel(panelScreenStyle, panelScreenRowH, items...).SetPadding(panelScreenPad)

	rect := menuframe.CenterWindowRect(world)
	x := rect.Min.X + (rect.Dx()-panelW)/2
	y := max(rect.Min.Y+(rect.Dy()-panelH)/2, rect.Min.Y)
	panel.Layout(image.Rect(x, y, x+panelW, y+panelH))
	return panel
}

// menuSelectedStyle は選択行の強調。背景を敷いてカーソル位置を示す。
var menuSelectedStyle = ui.BoxStyle{Fill: theme.PanelHighlight}

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
func renderMenuListUI(itemIndex int, rows []menuRow, colWidths []int, aligns []styled.TextAlign, opts menuListOpts, face text.Face) []ui.Widget {
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
		items = append(items, dataRowUI(entry.Item.Cells, colWidths, aligns, pg.IsSelectedInPage(entry.Index), face))
	}
	// 複数ページの画面は各ページを1ページ件数ぶんの空行で埋め、ページを繰っても高さを一定にする
	if len(rows) > perPage {
		for i := len(visible); i < perPage; i++ {
			items = append(items, blankRowUI(colWidths, face))
		}
	}
	return items
}

// headerRowUI は見出し行を組む。カーソルは止まらず、補助色で描く。
func headerRowUI(texts []string, colWidths []int, face text.Face) *ui.Container {
	cells := make([]ui.Widget, len(texts))
	for i, s := range texts {
		cells[i] = ui.NewText(s, face, theme.TextSecondary)
	}
	return ui.Row(colWidths, cells...)
}

// dataRowUI はデータ行を組む。選択中なら背景を敷き文字色を選択色にする。アイコンセルは画像で描く。
func dataRowUI(cells []styled.Cell, colWidths []int, aligns []styled.TextAlign, selected bool, face text.Face) *ui.Container {
	var textColor color.Color = theme.TextPrimary
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
		if i < len(aligns) {
			t.Align = toUIAlign(aligns[i])
		}
		cellWidgets[i] = t
	}
	row := ui.Row(colWidths, cellWidgets...)
	if selected {
		row.SetStyle(menuSelectedStyle)
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
