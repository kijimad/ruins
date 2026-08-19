// Package menuinput はメニュー操作のキーボード入力を Action へ変換し、再生時は world の
// 供給源へ差し替える。menuloop と overlay の両方から使い、入力層の差し替え点を1つにする。
// menuloop は overlay を import するため overlay から menuloop は呼べない。両者より下の
// このパッケージに置くことで、循環せず入力経路を共有できる。
package menuinput

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
)

// ReadMenuInput は1フレームぶんのメニュー操作を Action として読む。world が入力供給源を
// 持つならそこから読み、持たない本番ではキーボードから変換する。ここで入れ替わるのは
// キー入力の有無だけで、後段の DoAction 以降は本番と完全に同じ経路を通る。
func ReadMenuInput(world w.World) (inputmapper.ActionID, bool) {
	if src := world.Resources.MenuInput; src != nil {
		return src()
	}
	return handleMenuInput()
}

// handleMenuInput はメニュー操作のキー入力を Action に変換する。本番の入力経路
func handleMenuInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}
	// 左右キーはタブ切替に固定する。全メニューで意味を揃え、ページ送りには使わない。
	// 長い一覧は上下でカーソルがページ境界を越えると自動でページが繰られる
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyArrowLeft) {
		return inputmapper.ActionMenuTabPrev, true
	}
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyArrowRight) {
		return inputmapper.ActionMenuTabNext, true
	}
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyArrowUp) {
		return inputmapper.ActionMenuUp, true
	}
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyArrowDown) {
		return inputmapper.ActionMenuDown, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyTab) {
		if keyboardInput.IsKeyPressed(ebiten.KeyShift) {
			return inputmapper.ActionMenuTabPrev, true
		}
		return inputmapper.ActionMenuTabNext, true
	}
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionMenuSelect, true
	}
	return "", false
}
