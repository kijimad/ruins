package menuscreen

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	w "github.com/kijimaD/ruins/internal/world"
)

// Action はアクション選択ウィンドウの1項目。Run が nil の項目は選ぶと窓を閉じるだけ
type Action struct {
	Label string
	Run   func(world w.World) error
}

// ActionWindow はアクション選択ウィンドウの表示状態・カーソル入力・実行・描画をまとめて担う。
// 呼び出し側は「見出しと選択肢、選ばれたときの実行」を返す provide 関数を渡すだけでよい。
// Enter で選択肢を実行して閉じ、Esc で閉じる。上下でカーソルを循環する。Detail と同じ使い方に揃える
type ActionWindow struct {
	active  bool
	index   int
	provide func(world w.World) (title string, actions []Action, ok bool)
}

// NewActionWindow は現在の対象に対する見出し・選択肢・実行を返す provide を受け取り ActionWindow を作る
func NewActionWindow(provide func(world w.World) (string, []Action, bool)) ActionWindow {
	return ActionWindow{provide: provide}
}

// Active はウィンドウを表示中かを返す
func (a *ActionWindow) Active() bool { return a.active }

// Open はウィンドウを先頭選択で開く
func (a *ActionWindow) Open() {
	a.active = true
	a.index = 0
}

// HandleInput は表示中のキー入力を処理する。UI を作り直すべきなら第1返り値を true にする。
// Enter で選択肢の Run を実行して閉じ、Esc で閉じる。Run のエラーはそのまま返す
func (a *ActionWindow) HandleInput(world w.World) error {
	if !a.active {
		return nil
	}
	ki := input.GetSharedKeyboardInput()
	_, actions, ok := a.provide(world)
	n := 0
	if ok {
		n = len(actions)
	}
	switch {
	case ki.IsKeyJustPressed(ebiten.KeyEscape):
		a.active = false
	case ki.IsEnterJustPressedOnce():
		a.active = false
		if ok && a.index >= 0 && a.index < n && actions[a.index].Run != nil {
			return actions[a.index].Run(world)
		}
	case ki.IsKeyPressedWithRepeat(ebiten.KeyArrowUp) && n > 0:
		a.index = (a.index - 1 + n) % n
	case ki.IsKeyPressedWithRepeat(ebiten.KeyArrowDown) && n > 0:
		a.index = (a.index + 1) % n
	}
	return nil
}

// Window は現在の選択肢からウィンドウを rect の位置へ組み立てる。対象が無ければ nil を返す
func (a *ActionWindow) Window(world w.World, rect image.Rectangle) *widget.Window {
	title, actions, ok := a.provide(world)
	if !ok {
		return nil
	}
	labels := make([]string, len(actions))
	for i, act := range actions {
		labels[i] = act.Label
	}
	return BuildActionWindow(world, rect, title, labels, a.index)
}
