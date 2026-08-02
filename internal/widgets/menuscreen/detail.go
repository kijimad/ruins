package menuscreen

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
)

// Detail は詳細モーダルの表示状態とページ送り入力を担う。
// x で開き、表示中は左右でページを繰り、Esc・x・Enter で閉じる。
// 表示内容の組み立ては呼び出し側が Active と Page を見て行う。
// これにより各メニュー state が詳細モーダルの入力分岐を個別に持たずに済む
type Detail struct {
	active bool
	page   int
}

// Active は詳細モーダルを表示中かを返す
func (d *Detail) Active() bool { return d.active }

// Page は現在の表示ページを返す
func (d *Detail) Page() int { return d.page }

// Open は詳細モーダルを先頭ページで開く
func (d *Detail) Open() {
	d.active = true
	d.page = 0
}

// HandleInput は表示中のキー入力を処理し、UI を作り直すべきなら true を返す。
// pageCount は現在の対象のページ数で、範囲外へはページを進めない。
// 表示中でないときは何もせず false を返す
func (d *Detail) HandleInput(pageCount int) bool {
	if !d.active {
		return false
	}
	ki := input.GetSharedKeyboardInput()
	switch {
	case ki.IsKeyJustPressed(ebiten.KeyEscape) || ki.IsKeyJustPressed(ebiten.KeyX) || ki.IsEnterJustPressedOnce():
		d.active = false
		return true
	case ki.IsKeyPressedWithRepeat(ebiten.KeyArrowLeft) && d.page > 0:
		d.page--
		return true
	case ki.IsKeyPressedWithRepeat(ebiten.KeyArrowRight) && d.page < pageCount-1:
		d.page++
		return true
	}
	return false
}
