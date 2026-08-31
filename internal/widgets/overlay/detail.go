package overlay

import (
	"image"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

var _ ScreenRenderer = (*Detail)(nil)

// DetailContent は詳細モーダルに出す1件分の内容。名前・説明・性能行をそのまま持つ。
// 実体から組むなら EntityDetailContent を使い、独自の行を出すなら Rows を直接与える。
// 行の型は entityspec.SpecRow を使う
type DetailContent struct {
	Name string
	Desc string
	Rows []entityspec.SpecRow
}

// EntityDetailContent は実体から名前・説明・性能行を組んだ詳細内容を返す。
// 在庫や持ち物のように対象の実体さえあれば表示を導ける詳細で使う。
// 死んだ実体には空を返し、ゼロ実体への Get で落ちるのを防ぐ
func EntityDetailContent(world w.World, e ecs.Entity) DetailContent {
	if !world.ECS.Alive(e) {
		return DetailContent{}
	}
	desc := ""
	if world.Components.Description.Has(e) {
		desc = query.T(world, world.Components.Description.Get(e).Description)
	}
	return DetailContent{
		Name: query.GetEntityName(e, world),
		Desc: desc,
		Rows: entityspec.SpecRows(world, e),
	}
}

// Detail は詳細モーダルの表示状態・ページ送り入力・ウィンドウ組み立てをまとめて担う。
// provide が返す内容を描き、1件なら中央に1枚、複数なら横並びで並べる。呼び出し側は
// 「何を出すか」を返す provide を渡すだけでよく、入力・ページ数・描画といった内部には触れない。
// x で開き、左右でページを繰り、Esc・Enter で閉じる
type Detail struct {
	active  bool
	page    int
	provide func(world w.World) ([]DetailContent, bool)
}

// NewDetail は1件の詳細内容を返す provide を受け取り Detail を作る。中央に1枚で描く。
// provide は対象が無ければ ok=false を返す
func NewDetail(provide func(world w.World) (DetailContent, bool)) Detail {
	return Detail{provide: func(world w.World) ([]DetailContent, bool) {
		c, ok := provide(world)
		if !ok {
			return nil, false
		}
		return []DetailContent{c}, true
	}}
}

// NewComparison は複数の詳細内容を返す provide を受け取り Detail を作る。返った内容を横並びで描く
func NewComparison(provide func(world w.World) ([]DetailContent, bool)) Detail {
	return Detail{provide: provide}
}

// NewEntityDetail は選択中の実体をそのまま詳細に出す Detail を作る。
// 対象の解決だけ渡せば、名前・説明・性能は実体から組む。対象が無いか死んでいれば開かない
func NewEntityDetail(provide func() (ecs.Entity, bool)) Detail {
	return NewDetail(func(world w.World) (DetailContent, bool) {
		e, ok := provide()
		if !ok || !world.ECS.Alive(e) {
			return DetailContent{}, false
		}
		return EntityDetailContent(world, e), true
	})
}

// Active は詳細モーダルを表示中かを返す
func (d *Detail) Active() bool { return d.active }

// Open は詳細モーダルを先頭ページで開く。出す内容が無ければ開かない。
// アイテムが無いメニューで開くと、中身の無いモーダルが入力を奪って抜けられなくなるのを防ぐ。
func (d *Detail) Open(world w.World) {
	if _, ok := d.provide(world); !ok {
		return
	}
	d.active = true
	d.page = 0
}

// HandleInput は表示中のキー入力を処理する。ページ数は provide の内容から自身で算出する。
// 表示中でなければ何もしない。
// error は Layer の契約に合わせた戻り値で、詳細モーダルでは常に nil
func (d *Detail) HandleInput(world w.World) error {
	if !d.active {
		return nil
	}
	// メニューと同じ入力供給源から読む。再生ドライバが差した Action もここに届くので、
	// overlay 表示中も本番フローのまま駆動できる
	action, ok := keybind.ReadInput(world, keybind.MenuCommon)
	if !ok {
		return nil
	}
	total := 1
	if contents, ok := d.provide(world); ok {
		// 複数枚は行数の最も多い枚に合わせてページ数を決める
		for _, c := range contents {
			if p := detailPageCount(len(c.Rows)); p > total {
				total = p
			}
		}
	}
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionMenuSelect:
		d.active = false
	case inputmapper.ActionMenuTabPrev, inputmapper.ActionMenuLeft:
		if d.page > 0 {
			d.page--
		}
	case inputmapper.ActionMenuTabNext, inputmapper.ActionMenuRight:
		if d.page < total-1 {
			d.page++
		}
	default:
		// 詳細モーダルは閉じるとページ送りしか扱わない
	}
	return nil
}

// RenderOverlay は現在の内容から詳細モーダルを uicore のツリーへ組み、rect 内に配置して返す。
// 対象が無ければ nil を返す。Screen はこれを本体の上へ重ねて描く。
func (d *Detail) RenderOverlay(world w.World, rect image.Rectangle) uicore.Drawable {
	contents, ok := d.provide(world)
	if !ok || len(contents) == 0 {
		return nil
	}
	res := world.Resources.UIResources
	if len(contents) == 1 {
		c := contents[0]
		return buildDetailUI(res, rect, c.Name, c.Desc, c.Rows, d.page)
	}
	// 複数枚は狭い WindowRect でなく画面幅の大半へ横並びで描く。上下は rect と揃える
	sw := world.Resources.ScreenDimensions.Width
	wide := image.Rect(theme.MenuModalMarginX, rect.Min.Y, sw-theme.MenuModalMarginX, rect.Max.Y)
	return buildPanelsUI(res, wide, contents, d.page)
}
