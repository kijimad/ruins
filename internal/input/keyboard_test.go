package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestGlobalKeyStateLogic(t *testing.T) {
	t.Parallel()
	// グローバルキー状態の動作テスト（Enter状態制御）

	// テスト用にリセット
	ResetGlobalKeyStateForTest()

	_ = NewDefaultKeyboardInput() // 異なるインスタンスでもグローバル状態を共有することを確認

	// 初期状態ではfalse
	if GlobalKeyState.IsInEnterPressSession() {
		t.Error("初期状態でEnterセッション状態がtrueになっている")
	}

	// keyboard1で模擬的にセッション状態設定
	GlobalKeyState.SetEnterPressSession(true)

	// keyboard2からも同じ状態が見える
	if !GlobalKeyState.IsInEnterPressSession() {
		t.Error("グローバル状態が正しく共有されていない")
	}

	// リセット機能の動作確認
	ResetGlobalKeyStateForTest()
	if GlobalKeyState.IsInEnterPressSession() {
		t.Error("リセット後にグローバル状態がクリアされていない")
	}
}

func TestMockKeyboardInput_EnterFunction(t *testing.T) {
	t.Parallel()
	// テスト用にグローバル状態をリセット
	ResetGlobalKeyStateForTest()

	mock := NewMockKeyboardInput()

	// Enterキー押下-押上機能のテスト
	// 1. 押下状態にする
	mock.SetKeyPressed(ebiten.KeyEnter, true)
	if mock.IsEnterJustPressedOnce() {
		t.Error("押下状態のみでは検出されるべきではない")
	}

	// 2. 押上状態にする（この時点で検出される）
	mock.SetKeyPressed(ebiten.KeyEnter, false)
	if !mock.IsEnterJustPressedOnce() {
		t.Error("押下-押上のワンセットが検出されなかった")
	}

	// 3. 再度同じ操作をしても検出されない（まだ押下していない状態）
	if mock.IsEnterJustPressedOnce() {
		t.Error("押上状態での連続検出が発生した")
	}

	// リセット後の確認
	mock.Reset()
	if mock.IsKeyPressed(ebiten.KeyEnter) {
		t.Error("Reset()後にキー状態がクリアされていない")
	}
}

func TestEnterKeyPressReleaseSequence(t *testing.T) {
	t.Parallel()
	// テスト用にグローバル状態をリセット
	ResetGlobalKeyStateForTest()

	// Enterキーの押下-押上シーケンステスト
	mock := NewMockKeyboardInput()

	// 1. 最初の押下-押上セット（成功）
	mock.SimulateEnterPressRelease()
	if !mock.IsEnterJustPressedOnce() {
		t.Error("初回の押下-押上セットが検出されなかった")
	}

	// 2. 2回目の押下-押上セット（成功）
	mock.SimulateEnterPressRelease()
	if !mock.IsEnterJustPressedOnce() {
		t.Error("2回目の押下-押上セットが検出されなかった")
	}

	// 3. 押下のみ（検出されない）
	mock.SetKeyPressed(ebiten.KeyEnter, true)
	if mock.IsEnterJustPressedOnce() {
		t.Error("押下のみで検出された")
	}

	// 4. さらに押下を続ける（検出されない）
	if mock.IsEnterJustPressedOnce() {
		t.Error("押下継続中に検出された")
	}

	// 5. 押上時に検出される
	mock.SetKeyPressed(ebiten.KeyEnter, false)
	if !mock.IsEnterJustPressedOnce() {
		t.Error("押上時に検出されなかった")
	}
}

func TestSharedKeyboardInput(t *testing.T) {
	t.Parallel()
	// 共有インスタンスのテスト

	// 複数回呼び出しても同じインスタンスが返される
	keyboard1 := GetSharedKeyboardInput()
	keyboard2 := GetSharedKeyboardInput()

	if keyboard1 != keyboard2 {
		t.Error("GetSharedKeyboardInput()が異なるインスタンスを返している")
	}

	// キーボード入力が正常に動作することを確認
	require.NotNil(t, keyboard1, "GetSharedKeyboardInput() returned nil")
}

func TestGlobalEnterPressStateStorage(t *testing.T) {
	t.Parallel()
	// テスト前にグローバル状態をリセット
	ResetGlobalKeyStateForTest()

	// 初期状態ではfalseであることを確認
	if GlobalKeyState.IsInEnterPressSession() {
		t.Error("初期状態でEnterセッション状態がtrueになっている")
	}

	// グローバルにセッション状態を設定
	GlobalKeyState.SetEnterPressSession(true)

	// グローバルからセッション状態を取得して確認
	if !GlobalKeyState.IsInEnterPressSession() {
		t.Error("グローバルのEnterセッション状態が正しく保存されていない")
	}

	// リセット機能のテスト
	ResetGlobalKeyStateForTest()
	if GlobalKeyState.IsInEnterPressSession() {
		t.Error("リセット後にEnterセッション状態がfalseでない")
	}
}
