package overlay

import (
	"fmt"
	"image"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// buildCompareUI は2件の詳細内容を左右に並べて rect 内へ組む。左に現装備、右に候補を
// 同じ密度で見せる。各枠は buildDetailUI と同じ組み方で、ページ番号も枠ごとに持つ。
func buildCompareUI(res resources.UIResources, rect image.Rectangle, left, right DetailContent, page int) uicore.Widget {
	gap := theme.Space7
	half := (rect.Dx() - gap) / 2
	leftRect := image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+half, rect.Max.Y)
	rightRect := image.Rect(rect.Max.X-half, rect.Min.Y, rect.Max.X, rect.Max.Y)
	lw := buildDetailUI(res, leftRect, left.Name, left.Desc, left.Rows, page)
	rw := buildDetailUI(res, rightRect, right.Name, right.Desc, right.Rows, page)
	group := uicore.NewGroup(lw, rw)
	group.Layout(rect)
	return group
}

// buildDetailUI は性能行の並びから詳細モーダルを uicore のツリーとして組み、rect いっぱいに配置して返す。
// name が空なら名前行を省き、desc が空なら説明行を省く。行が多いときは page でページ分割する。
// 説明は最終ページにだけ出す。位置表示は1ページでも常に出す。page は範囲外なら内部でクランプする。
// 背景はパネルテクスチャを rect 全体へ敷き、内容は上寄せにする。
func buildDetailUI(res resources.UIResources, rect image.Rectangle, name, desc string, rows []entityspec.SpecRow, page int) uicore.Widget {
	face := res.Text.BodyFace
	// 説明とページ番号は小さめのフェイスで補助色、名前と性能行は本文フェイスにする。
	smallFace := res.Text.SmallFace
	// 行高は本文フェイスの行送りにする。全行を同じ高さで並べ、いちばん背の高い本文が字面を切らない。
	rowH := uicore.LineHeight(face)

	total := detailPageCount(len(rows))
	if page < 0 {
		page = 0
	}
	if page >= total {
		page = total - 1
	}
	pg := pagination.New(page*detailRowsPerPage, len(rows), detailRowsPerPage)
	start, end := pg.GetVisibleRange()

	var items []uicore.Widget
	if name != "" {
		items = append(items, uicore.NewText(name, face, theme.TextPrimary))
	}
	items = append(items, entityspec.SpecRowWidgets(rows[start:end], face)...)
	if desc != "" && page == total-1 {
		for _, line := range uicore.WrapText(desc, smallFace, rect.Dx()-theme.Space7*2) {
			items = append(items, uicore.NewText(line, smallFace, theme.TextSecondary))
		}
	}
	items = append(items, uicore.NewText(fmt.Sprintf("%d/%d", pg.GetCurrentPage(), pg.GetTotalPages()), smallFace, theme.TextSecondary))

	// 内側余白はパネルテクスチャの枠を避けて内容を内側へ寄せる
	panel := uicore.VBox(rowH, items...).SetPadding(theme.Space7)
	if res.PanelBG != nil {
		panel.SetBackgroundNineSlice(res.PanelBG.Image, res.PanelBG.BX, res.PanelBG.BY)
	}
	panel.Layout(rect)
	return panel
}
