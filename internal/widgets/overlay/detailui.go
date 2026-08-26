package overlay

import (
	"fmt"
	"image"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

const (
	// detailModalWidth は詳細モーダルの幅。
	detailModalWidth = 240
	// detailModalPad はモーダルの内側余白。
	detailModalPad = 8
)

// detailModalStyle はモーダルの背景と枠。
var detailModalStyle = ui.BoxStyle{Fill: theme.WindowBackground, Border: theme.PanelHighlight, BorderWidth: 1}

// buildDetailUI は性能行の並びから詳細モーダルを internal/ui のツリーとして組み、rect 内に配置して返す。
// name が空なら名前行を省き、desc が空なら説明行を省く。行が多いときは page でページ分割する。
// 説明は最終ページにだけ出す。位置表示は1ページでも常に出す。page は範囲外なら内部でクランプする。
func buildDetailUI(face text.Face, rect image.Rectangle, name, desc string, rows []entityspec.SpecRow, page int) ui.Widget {
	rowH := entityspec.SpecPanelRowH

	total := detailPageCount(len(rows))
	if page < 0 {
		page = 0
	}
	if page >= total {
		page = total - 1
	}
	pg := pagination.New(page*detailRowsPerPage, len(rows), detailRowsPerPage)
	start, end := pg.GetVisibleRange()

	var items []ui.Widget
	if name != "" {
		items = append(items, ui.NewText(name, face, theme.TextPrimary))
	}
	items = append(items, entityspec.SpecRowWidgets(rows[start:end], face)...)
	if desc != "" && page == total-1 {
		for _, line := range ui.WrapText(desc, face, detailModalWidth-detailModalPad*2) {
			items = append(items, ui.NewText(line, face, theme.TextSecondary))
		}
	}
	items = append(items, ui.NewText(fmt.Sprintf("%d/%d", pg.GetCurrentPage(), pg.GetTotalPages()), face, theme.TextSecondary))

	panel := ui.Panel(detailModalStyle, rowH, items...).SetPadding(detailModalPad)
	height := len(items)*rowH + detailModalPad*2
	x := rect.Min.X + (rect.Dx()-detailModalWidth)/2
	y := rect.Min.Y
	panel.Layout(image.Rect(x, y, x+detailModalWidth, y+height))
	return panel
}
