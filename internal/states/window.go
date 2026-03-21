package states

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
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyArrowLeft) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyA) {
		return inputmapper.ActionMenuLeft, true
	}
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyArrowRight) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyD) {
		return inputmapper.ActionMenuRight, true
	}
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyArrowUp) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyW) {
		return inputmapper.ActionMenuUp, true
	}
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyArrowDown) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyS) {
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

// HandleWindowInput はウィンドウモード時のキー入力をActionに変換する
func HandleWindowInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	// 上移動キー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowUp) || keyboardInput.IsKeyJustPressed(ebiten.KeyW) {
		return inputmapper.ActionWindowUp, true
	}
	// 下移動キー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowDown) || keyboardInput.IsKeyJustPressed(ebiten.KeyS) {
		return inputmapper.ActionWindowDown, true
	}
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionWindowConfirm, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionWindowCancel, true
	}

	return "", false
}
