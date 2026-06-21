package input

import (
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// KeyRepeatInitialDelay はキーリピート開始までの遅延
	KeyRepeatInitialDelay = 500 * time.Millisecond

	// KeyRepeatInterval はキーリピートの間隔
	KeyRepeatInterval = 50 * time.Millisecond
)

// keyRepeatState はキーリピート状態を管理する
type keyRepeatState struct {
	pressStartTime time.Time // キーが押され始めた時刻
	lastRepeatTime time.Time // 最後にリピートした時刻
}

// KeyboardInput はキーボード入力を抽象化するインターフェース
type KeyboardInput interface {
	IsKeyJustPressed(key ebiten.Key) bool
	IsKeyPressed(key ebiten.Key) bool
	IsEnterJustPressedOnce() bool               // Enterキーが押下-押上のワンセットで押されたかどうか
	IsKeyPressedWithRepeat(key ebiten.Key) bool // キーリピート機能付きの押下判定
}

// sharedKeyboardInput はシングルトンのキーボード入力実装。
// キー状態をインスタンスフィールドとして保持する
type sharedKeyboardInput struct {
	enterPressSession bool                           // Enterキーの押下セッション状態
	keyRepeatStates   map[ebiten.Key]*keyRepeatState // キーリピート状態
	mu                sync.Mutex
}

var (
	keyboardInstance KeyboardInput
	once             sync.Once
)

// GetSharedKeyboardInput は共有されるキーボード入力インスタンスを返す
func GetSharedKeyboardInput() KeyboardInput {
	once.Do(func() {
		keyboardInstance = &sharedKeyboardInput{
			keyRepeatStates: make(map[ebiten.Key]*keyRepeatState),
		}
	})
	return keyboardInstance
}

func (s *sharedKeyboardInput) IsKeyJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

func (s *sharedKeyboardInput) IsKeyPressed(key ebiten.Key) bool {
	return ebiten.IsKeyPressed(key)
}

// IsEnterJustPressedOnce はEnterキーが押下-押上のワンセットで押されたかどうかを返す
func (s *sharedKeyboardInput) IsEnterJustPressedOnce() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentlyPressed := ebiten.IsKeyPressed(ebiten.KeyEnter)
	wasInSession := s.enterPressSession

	s.enterPressSession = currentlyPressed

	// セッション終了時（押下から押上への遷移）のみtrueを返す
	return wasInSession && !currentlyPressed
}

// IsKeyPressedWithRepeat はキーリピート機能付きの押下判定を行う
func (s *sharedKeyboardInput) IsKeyPressedWithRepeat(key ebiten.Key) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 初回押下
	if inpututil.IsKeyJustPressed(key) {
		now := time.Now()
		s.keyRepeatStates[key] = &keyRepeatState{
			pressStartTime: now,
			lastRepeatTime: now,
		}
		return true
	}

	// キーが押され続けている場合
	if ebiten.IsKeyPressed(key) {
		state, exists := s.keyRepeatStates[key]
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
	delete(s.keyRepeatStates, key)
	return false
}
