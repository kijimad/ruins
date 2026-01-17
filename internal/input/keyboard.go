package input

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// KeyRepeatInitialDelay はキーリピート開始までの遅延（ミリ秒）
	KeyRepeatInitialDelay = 500 * time.Millisecond

	// KeyRepeatInterval はキーリピートの間隔（ミリ秒）
	KeyRepeatInterval = 100 * time.Millisecond
)

// GlobalKeyState はグローバルなキー状態を管理する
var GlobalKeyState = &globalKeyState{
	enterPressSession: false,
	keyRepeatStates:   make(map[ebiten.Key]*keyRepeatState),
}

// keyRepeatState はキーリピート状態を管理する
type keyRepeatState struct {
	pressStartTime time.Time // キーが押され始めた時刻
	lastRepeatTime time.Time // 最後にリピートした時刻
}

// globalKeyState はアプリケーション全体で共有されるキー状態
type globalKeyState struct {
	enterPressSession bool                           // Enterキーの押下セッション状態（押下中かどうか）
	keyRepeatStates   map[ebiten.Key]*keyRepeatState // キーリピート状態
	mutex             sync.RWMutex
}

// SetEnterPressSession はEnterキーの押下セッション状態を設定する
func (g *globalKeyState) SetEnterPressSession(inSession bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.enterPressSession = inSession
}

// IsInEnterPressSession はEnterキーが押下セッション中かどうかを取得する
func (g *globalKeyState) IsInEnterPressSession() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.enterPressSession
}

// ResetGlobalKeyStateForTest はテスト用にグローバルキー状態をリセットする
func ResetGlobalKeyStateForTest() {
	GlobalKeyState.mutex.Lock()
	defer GlobalKeyState.mutex.Unlock()
	GlobalKeyState.enterPressSession = false // Enterキーセッション状態をリセット
}

// KeyboardInput はキーボード入力を抽象化するインターフェース
type KeyboardInput interface {
	IsKeyJustPressed(key ebiten.Key) bool
	IsKeyPressed(key ebiten.Key) bool
	IsEnterJustPressedOnce() bool               // Enterキーが押下-押上のワンセットで押されたかどうか
	IsKeyPressedWithRepeat(key ebiten.Key) bool // キーリピート機能付きの押下判定
}

// sharedKeyboardInput はシングルトンのキーボード入力実装
type sharedKeyboardInput struct{}

var (
	// keyboardInstance は共有されるキーボード入力インスタンス
	keyboardInstance KeyboardInput
	once             sync.Once
)

// GetSharedKeyboardInput は共有されるキーボード入力インスタンスを返す
func GetSharedKeyboardInput() KeyboardInput {
	once.Do(func() {
		keyboardInstance = &sharedKeyboardInput{}
	})
	return keyboardInstance
}

// NewDefaultKeyboardInput は新しいデフォルトキーボード入力インスタンスを返す
func NewDefaultKeyboardInput() KeyboardInput {
	return &sharedKeyboardInput{}
}

func (s *sharedKeyboardInput) IsKeyJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

func (s *sharedKeyboardInput) IsKeyPressed(key ebiten.Key) bool {
	return ebiten.IsKeyPressed(key)
}

// IsEnterJustPressedOnce はEnterキーが押下-押上のワンセットで押されたかどうかを返す
func (s *sharedKeyboardInput) IsEnterJustPressedOnce() bool {
	// 現在のEnterキーの物理状態を取得
	currentlyPressed := ebiten.IsKeyPressed(ebiten.KeyEnter)
	wasInSession := GlobalKeyState.IsInEnterPressSession()

	// セッション状態を更新
	GlobalKeyState.SetEnterPressSession(currentlyPressed)

	// セッション終了時（押下から押上への遷移）のみtrueを返す（1セット完了）
	if wasInSession && !currentlyPressed {
		return true
	}

	return false
}

// IsKeyPressedWithRepeat はキーリピート機能付きの押下判定を行う
func (s *sharedKeyboardInput) IsKeyPressedWithRepeat(key ebiten.Key) bool {
	GlobalKeyState.mutex.Lock()
	defer GlobalKeyState.mutex.Unlock()

	// 初回押下
	if inpututil.IsKeyJustPressed(key) {
		now := time.Now()
		GlobalKeyState.keyRepeatStates[key] = &keyRepeatState{
			pressStartTime: now,
			lastRepeatTime: now,
		}
		return true
	}

	// キーが押され続けている場合
	if ebiten.IsKeyPressed(key) {
		state, exists := GlobalKeyState.keyRepeatStates[key]
		if !exists {
			return false
		}

		now := time.Now()
		pressDuration := now.Sub(state.pressStartTime)

		// 初回遅延未経過
		if pressDuration < KeyRepeatInitialDelay {
			return false
		}

		// リピート間隔チェック
		timeSinceLastRepeat := now.Sub(state.lastRepeatTime)
		if timeSinceLastRepeat >= KeyRepeatInterval {
			state.lastRepeatTime = now
			return true
		}

		return false
	}

	// キーが離された場合、状態をクリア
	delete(GlobalKeyState.keyRepeatStates, key)
	return false
}
