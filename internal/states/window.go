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
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowLeft) || keyboardInput.IsKeyJustPressed(ebiten.KeyA) {
		return inputmapper.ActionMenuLeft, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowRight) || keyboardInput.IsKeyJustPressed(ebiten.KeyD) {
		return inputmapper.ActionMenuRight, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowUp) || keyboardInput.IsKeyJustPressed(ebiten.KeyW) {
		return inputmapper.ActionMenuUp, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowDown) || keyboardInput.IsKeyJustPressed(ebiten.KeyS) {
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

// UpdateFocusIndex はナビゲーションアクションに応じてフォーカスインデックスを更新する
func UpdateFocusIndex(action inputmapper.ActionID, focusIndex *int, itemCount int) bool {
	switch action {
	case inputmapper.ActionWindowUp:
		*focusIndex--
		if *focusIndex < 0 {
			*focusIndex = itemCount - 1
		}
		return true
	case inputmapper.ActionWindowDown:
		*focusIndex++
		if *focusIndex >= itemCount {
			*focusIndex = 0
		}
		return true
	default:
		return false
	}
}
