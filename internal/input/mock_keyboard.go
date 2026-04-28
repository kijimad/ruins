package input

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// MockKeyboardInput はテスト用のモックキーボード入力実装
type MockKeyboardInput struct {
	pressedKeys           map[ebiten.Key]bool
	justPressedKeys       map[ebiten.Key]bool
	pressedWithRepeatKeys map[ebiten.Key]bool
	previousEnterSession  bool // 前回のEnterキーセッション状態
}

// NewMockKeyboardInput はモックキーボード入力を作成する
func NewMockKeyboardInput() *MockKeyboardInput {
	return &MockKeyboardInput{
		pressedKeys:           make(map[ebiten.Key]bool),
		justPressedKeys:       make(map[ebiten.Key]bool),
		pressedWithRepeatKeys: make(map[ebiten.Key]bool),
		previousEnterSession:  false,
	}
}

// IsKeyPressed はキーが現在押されているかを返す
func (m *MockKeyboardInput) IsKeyPressed(key ebiten.Key) bool {
	return m.pressedKeys[key]
}

// SetKeyPressed はテスト用にキーの状態を設定する
func (m *MockKeyboardInput) SetKeyPressed(key ebiten.Key, pressed bool) {
	m.pressedKeys[key] = pressed
}

// IsKeyJustPressed はキーが今フレームで初めて押されたかを返す
func (m *MockKeyboardInput) IsKeyJustPressed(key ebiten.Key) bool {
	return m.justPressedKeys[key]
}

// SetKeyJustPressed はテスト用にキーのJustPressed状態を設定する
func (m *MockKeyboardInput) SetKeyJustPressed(key ebiten.Key, pressed bool) {
	m.justPressedKeys[key] = pressed
}

// IsKeyPressedWithRepeat はキーリピート付きの押下判定を返す
func (m *MockKeyboardInput) IsKeyPressedWithRepeat(key ebiten.Key) bool {
	return m.pressedWithRepeatKeys[key]
}

// SetKeyPressedWithRepeat はテスト用にキーのリピート付き押下状態を設定する
func (m *MockKeyboardInput) SetKeyPressedWithRepeat(key ebiten.Key, pressed bool) {
	m.pressedWithRepeatKeys[key] = pressed
}

// Reset は全てのキー状態をリセットする
func (m *MockKeyboardInput) Reset() {
	m.pressedKeys = make(map[ebiten.Key]bool)
	m.justPressedKeys = make(map[ebiten.Key]bool)
	m.pressedWithRepeatKeys = make(map[ebiten.Key]bool)
	m.previousEnterSession = false
}

// IsEnterJustPressedOnce はモック用の押下-押上ワンセット検出
func (m *MockKeyboardInput) IsEnterJustPressedOnce() bool {
	// 現在のEnterキーセッション状態を取得
	currentlyInSession := m.pressedKeys[ebiten.KeyEnter]
	wasInSession := m.previousEnterSession

	// セッション状態を更新
	m.previousEnterSession = currentlyInSession

	// セッション終了時（押下から押上への遷移）のみtrueを返す
	if wasInSession && !currentlyInSession {
		return true
	}

	return false
}

// SimulateEnterPressRelease はテスト用にEnterキーの押下-押上をシミュレートする
func (m *MockKeyboardInput) SimulateEnterPressRelease() {
	// セッション開始（押下状態に設定）
	m.SetKeyPressed(ebiten.KeyEnter, true)
	m.IsEnterJustPressedOnce() // セッション状態を更新

	// セッション終了（押上状態に設定）
	m.SetKeyPressed(ebiten.KeyEnter, false)
	// この時点でIsEnterJustPressedOnce()がtrueを返すはず（1セット完了）
}
