package menuscreen

import (
	"fmt"
	"image"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// detailRowsPerPage は詳細ウィンドウ1ページに収める性能行の数。
// 行数でページ分割することで、短い項目は1ページに収まり、行の多い項目だけがはみ出さないよう分割される
const detailRowsPerPage = 12

// WindowCursorReducer は選択肢ウィンドウの上下カーソル移動を扱う reducer を返す。
// 端では循環し、選択肢が無いときは 0 に留まる。各メニューのウィンドウで共通に使う
func WindowCursorReducer(count int) func(int, inputmapper.ActionID) int {
	return func(v int, action inputmapper.ActionID) int {
		if count == 0 {
			return 0
		}
		switch action {
		case inputmapper.ActionWindowUp:
			return (v - 1 + count) % count
		case inputmapper.ActionWindowDown:
			return (v + 1) % count
		default:
			return v
		}
	}
}

// BuildActionWindow は選択肢を縦に並べるサブウィンドウを組み立て、rect の位置に置く。
// selectedIndex の行にカーソルを表示する。title が空でもヘッダ帯は描かれる
func BuildActionWindow(world w.World, rect image.Rectangle, title string, actions []string, selectedIndex int) *widget.Window {
	res := world.Resources.UIResources
	content := styled.NewWindowContainer(res)
	header := styled.NewWindowHeaderContainer(title, res)
	window := styled.NewSmallWindow(header, content)
	for i, action := range actions {
		content.AddChild(styled.NewListItemText(action, theme.TextSecondary, i == selectedIndex, res))
	}
	window.SetLocation(rect)
	return window
}

// detailPageCount は行数からページ数を返す。行が無くても1を返す
func detailPageCount(rowCount int) int {
	if rowCount <= 0 {
		return 1
	}
	return (rowCount + detailRowsPerPage - 1) / detailRowsPerPage
}

// DetailPageCount はエンティティの詳細ウィンドウのページ数を返す
func DetailPageCount(world w.World, entity ecs.Entity) int {
	return detailPageCount(len(views.SpecRows(world, entity)))
}

// buildDetailFromRows は性能行の並びから詳細ウィンドウを組み立てる。
// name が空なら名前行を省き、desc が空なら説明行を省く。タイトルバーは表示しない。
// 行が多いときは page でページ分割する。位置表示は1ページでも常に出し、ページ有無で
// 位置がずれないようにする。page は範囲外なら内部でクランプする
func buildDetailFromRows(world w.World, rect image.Rectangle, name, desc string, rows []views.SpecRow, page int) *widget.Window {
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
	start := page * detailRowsPerPage
	end := min(start+detailRowsPerPage, len(rows))

	spec := styled.NewVerticalContainer()
	views.RenderSpecRows(spec, rows[start:end], res)
	content.AddChild(spec)

	// 位置表示は1ページでも常設し、ページ有無で表示がずれないようにする。左右キーでページを繰る
	content.AddChild(styled.NewDescriptionText(fmt.Sprintf("%s %d/%d %s", consts.IconArrowLeft, page+1, total, consts.IconArrowRight), res))
	if desc != "" {
		content.AddChild(styled.NewDescriptionText(desc, res))
	}
	win := styled.NewSmallWindow(widget.NewContainer(), content)
	win.SetLocation(rect)
	return win
}
