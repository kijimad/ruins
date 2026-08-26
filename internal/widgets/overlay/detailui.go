package overlay

import (
	"fmt"
	"image"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

// detailModalPad はモーダルの内側余白。パネルテクスチャの枠を避けて内容を内側へ寄せる。
// 元の NewWindowContainer の上下の余白 Space7 に合わせる。
const detailModalPad = theme.Space7

// buildDetailUI は性能行の並びから詳細モーダルを internal/ui のツリーとして組み、rect いっぱいに配置して返す。
// name が空なら名前行を省き、desc が空なら説明行を省く。行が多いときは page でページ分割する。
// 説明は最終ページにだけ出す。位置表示は1ページでも常に出す。page は範囲外なら内部でクランプする。
// 背景はパネルテクスチャを rect 全体へ敷き、内容は上寄せにする。元の overlay ウィンドウと同じ意匠。
func buildDetailUI(res resources.UIResources, rect image.Rectangle, name, desc string, rows []entityspec.SpecRow, page int) ui.Widget {
	face := res.Text.BodyFace
	// 説明とページ番号は元の NewDescriptionText と同じ小さめのフェイスで補助色にする。
	// 名前と性能行は本文フェイス。説明を小フェイスにすると幅に収まり折り返さない。
	smallFace := res.Text.SmallFace
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
		for _, line := range ui.WrapText(desc, smallFace, rect.Dx()-detailModalPad*2) {
			items = append(items, ui.NewText(line, smallFace, theme.TextSecondary))
		}
	}
	items = append(items, ui.NewText(fmt.Sprintf("%d/%d", pg.GetCurrentPage(), pg.GetTotalPages()), smallFace, theme.TextSecondary))

	panel := ui.VBox(rowH, items...).SetPadding(detailModalPad)
	if res.PanelBG != nil {
		panel.SetBackgroundNineSlice(res.PanelBG.Image, res.PanelBG.BX, res.PanelBG.BY)
	}
	panel.Layout(rect)
	return panel
}
