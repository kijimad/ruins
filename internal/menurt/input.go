package menurt

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
)

// HandleMenuInput はメニュー操作のキー入力をActionに変換する
func HandleMenuInput() (inputmapper.ActionID, bool) {
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
