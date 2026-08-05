package menuscreen

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// SpecRow は詳細モーダルの1行。情報タブなど spec 由来でない詳細を組む際に呼び出し側が使う
type SpecRow = views.SpecRow

// DetailContent は詳細モーダルに出す1件分の内容。中身は次のいずれかで指定する。
// Rows を与えればそれを使う。無ければ Spec、無ければ Entity から性能行を解決する。
// 呼び出し側は名前・説明と対象だけを渡し、行の生成やページ計算には触れない
type DetailContent struct {
	Name   string
	Desc   string
	Entity ecs.Entity     // エンティティ由来の性能
	Spec   *gc.EntitySpec // raw 定義由来の性能。商店の購入品など未生成の対象に使う
	Rows   []SpecRow      // 明示的な行。情報タブなど spec でない詳細に使う
}

// resolveRows は内容から表示行を解決する。Rows 優先、次に Spec、最後に Entity
func (c DetailContent) resolveRows(world w.World) []SpecRow {
	switch {
	case c.Rows != nil:
		return c.Rows
	case c.Spec != nil:
		return views.SpecRowsFromSpec(world, *c.Spec)
	default:
		return views.SpecRows(world, c.Entity)
	}
}

// Detail は詳細モーダルの表示状態・ページ送り入力・ウィンドウ組み立てをまとめて担う。
// 呼び出し側は「何を出すか」を返す provide 関数を渡すだけでよく、
// 入力・ページ数・描画といった内部には触れない。x で開き、左右でページを繰り、Esc・x・Enter で閉じる
type Detail struct {
	active  bool
	page    int
	provide func(world w.World) (DetailContent, bool)
}

// NewDetail は現在カーソルが指す対象の詳細内容を返す provide を受け取り Detail を作る。
// provide は対象が無ければ ok=false を返す
func NewDetail(provide func(world w.World) (DetailContent, bool)) Detail {
	return Detail{provide: provide}
}

// Active は詳細モーダルを表示中かを返す
func (d *Detail) Active() bool { return d.active }

// Open は詳細モーダルを先頭ページで開く
func (d *Detail) Open() {
	d.active = true
	d.page = 0
}

// HandleInput は表示中のキー入力を処理する。ページ数は provide の内容から自身で算出する。
// 表示中でなければ何もしない。
// error は Overlay インターフェースに合わせた形で、詳細モーダルでは常に nil
func (d *Detail) HandleInput(world w.World) error {
	if !d.active {
		return nil
	}
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyEscape) || ki.IsKeyJustPressed(ebiten.KeyX) || ki.IsEnterJustPressedOnce() {
		d.active = false
		return nil
	}
	total := 1
	if content, ok := d.provide(world); ok {
		total = detailPageCount(len(content.resolveRows(world)))
	}
	switch {
	case ki.IsKeyPressedWithRepeat(ebiten.KeyArrowLeft) && d.page > 0:
		d.page--
	case ki.IsKeyPressedWithRepeat(ebiten.KeyArrowRight) && d.page < total-1:
		d.page++
	}
	return nil
}

// Window は現在の内容から詳細ウィンドウを rect の位置へ組み立てる。対象が無ければ nil を返す
func (d *Detail) Window(world w.World, rect image.Rectangle) *widget.Window {
	content, ok := d.provide(world)
	if !ok {
		return nil
	}
	return buildDetailFromRows(world, rect, content.Name, content.Desc, content.resolveRows(world), d.page)
}
