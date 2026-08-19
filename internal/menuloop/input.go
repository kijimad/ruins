package menuloop

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
)

// ReadMenuInput は1フレームぶんのメニュー操作を Action として読む。world が入力供給源を
// 持つならそこから読み、持たない本番ではキーボードから変換する。ここが本番と再生で
// 唯一分岐する点で、キー入力の有無だけが入れ替わり、後段の DoAction 以降は完全に同じ経路を通る
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
