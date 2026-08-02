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

// detailSectionsPerPage は詳細ウィンドウ1ページに収める性能区画の数。
// component が多いアイテムでモーダルからはみ出さないよう区画単位でページ分割する
const detailSectionsPerPage = 3

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

// DetailPageCount は詳細ウィンドウのページ数を返す。性能区画が無くても1を返す
func DetailPageCount(world w.World, entity ecs.Entity) int {
	n := len(views.SpecSections(world, entity))
	if n <= 0 {
		return 1
	}
	return (n + detailSectionsPerPage - 1) / detailSectionsPerPage
}

// BuildDetailWindow はアイテムや装備の詳細ウィンドウを組み立て、rect の位置に置く。
// name が空なら名前行を省き、desc が空なら説明行を省く。タイトルバーは表示しない。
// 性能区画が多いときは page でページ分割し、複数ページなら位置表示を出す。
// page は範囲外なら内部でクランプする
func BuildDetailWindow(world w.World, rect image.Rectangle, name, desc string, entity ecs.Entity, page int) *widget.Window {
	res := world.Resources.UIResources
	content := styled.NewWindowContainer(res)
	if name != "" {
		content.AddChild(styled.NewMenuText(name, res))
	}

	sections := views.SpecSections(world, entity)
	total := DetailPageCount(world, entity)
	if page < 0 {
		page = 0
	}
	if page >= total {
		page = total - 1
	}
	start := page * detailSectionsPerPage
	end := min(start+detailSectionsPerPage, len(sections))

	spec := styled.NewVerticalContainer()
	views.RenderSpecSections(spec, sections[start:end])
	content.AddChild(spec)

	// 複数ページあるときだけ位置を示す。左右キーでページを繰る
	if total > 1 {
		content.AddChild(styled.NewDescriptionText(fmt.Sprintf("%s %d/%d %s", consts.IconArrowLeft, page+1, total, consts.IconArrowRight), res))
	}
	if desc != "" {
		content.AddChild(styled.NewDescriptionText(desc, res))
	}
	win := styled.NewSmallWindow(widget.NewContainer(), content)
	win.SetLocation(rect)
	return win
}
