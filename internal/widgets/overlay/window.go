package overlay

import (
	"fmt"
	"image"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// detailRowsPerPage は詳細ウィンドウ1ページに収める性能行の数。
// 行数でページ分割することで、短い項目は1ページに収まり、行の多い項目だけがはみ出さないよう分割される
const detailRowsPerPage = 12

// DetailPageCount はエンティティの詳細ウィンドウのページ数を返す。
// 呼び出し側が実体からページ数を確かめる公開の入口
func DetailPageCount(world w.World, entity ecs.Entity) int {
	return detailPageCount(len(entityspec.SpecRows(world, entity)))
}

// detailPageCount は行数からページ数を返す。ページ計算は pagination に委ねる。
// 行が無いか負数のときの1丸めも pagination.GetTotalPages が吸収する
func detailPageCount(rowCount int) int {
	return pagination.New(0, rowCount, detailRowsPerPage).GetTotalPages()
}

// buildDetailFromRows は性能行の並びから詳細ウィンドウを組み立てる。
// name が空なら名前行を省き、desc が空なら説明行を省く。タイトルバーは表示しない。
// 行が多いときは page でページ分割する。位置表示は1ページでも常に出し、ページ有無で
// 位置がずれないようにする。page は範囲外なら内部でクランプする
func buildDetailFromRows(world w.World, rect image.Rectangle, name, desc string, rows []entityspec.SpecRow, page int) *widget.Window {
	res := world.Resources.UIResources
	content := styled.NewWindowContainer(res)
	if name != "" {
		content.AddChild(styled.NewMenuText(name, res))
	}

	total := detailPageCount(len(rows))
	if page < 0 {
		page = 0
	}
	if page >= total {
		page = total - 1
	}
	// page を先頭アイテム位置に読み替えて pagination に可視範囲を委ねる
	pg := pagination.New(page*detailRowsPerPage, len(rows), detailRowsPerPage)
	start, end := pg.GetVisibleRange()

	spec := styled.NewVerticalContainer()
	entityspec.RenderSpecRows(spec, rows[start:end], res)
	content.AddChild(spec)

	// 説明は性能行のページ送りとは別物なので、最終ページのページャ直上にだけ出す。全ページへの重複を避ける
	if desc != "" && page == total-1 {
		content.AddChild(styled.NewDescriptionText(desc, res))
	}
	// 位置は番号だけで示す。矢印は付けない。全メニューのページ表示を番号だけに揃える。
	// 1ページでも 1/1 を常設し、ページ有無で表示がずれないようにする。左右キーでページを繰る。
	// GetPageText は単一ページで空を返すため、常設したいここでは GetCurrentPage/GetTotalPages を使う
	content.AddChild(styled.NewDescriptionText(fmt.Sprintf("%d/%d", pg.GetCurrentPage(), pg.GetTotalPages()), res))
	win := styled.NewSmallWindow(widget.NewContainer(), content)
	win.SetLocation(rect)
	return win
}
