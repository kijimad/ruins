package states

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/inputmapper"
)

// TransType is a transition type
type TransType int

const (
	// TransNone does nothing
	TransNone TransType = iota
	// TransPop removes the active state and resume the next state
	TransPop
	// TransPush pauses the active state and add new states to the stack
	TransPush
	// TransSwitch removes the active state and replace it by a new one
	TransSwitch
	// TransReplace removes all states and insert a new stack
	TransReplace
	// TransQuit removes all states and quit
	TransQuit
)

// Transition は状態遷移を表す
type Transition[T any] struct {
	Type          TransType
	NewStateFuncs []StateFactory[T]
}

// State はゲームステートのジェネリックインターフェース
type State[T any] interface {
	// Executed when the state begins
	OnStart(world T) error
	// Executed when the state exits
	OnStop(world T) error
	// Executed when a new state is pushed over this one
	OnPause(world T) error
	// Executed when the state become active again (states pushed over this one have been popped)
	OnResume(world T) error
	// Executed on every frame when the state is active
	Update(world T) (Transition[T], error)
	// 描画
	Draw(world T, screen *ebiten.Image) error
}

// ActionHandler はActionベースの入力処理を行うステートのためのオプショナルインターフェース
//   - HandleInput でキー入力をActionIDに変換
//   - DoAction でActionIDを受け取ってステート遷移を返す
type ActionHandler[T any] interface {
	// HandleInput はキー入力をActionIDに変換する
	HandleInput(cfg *config.Config) (inputmapper.ActionID, bool)

	// DoAction はActionを実行してステート遷移を返す
	DoAction(world T, action inputmapper.ActionID) (Transition[T], error)
}

// StateFactory はステートを作成するファクトリー関数の型
type StateFactory[T any] func() State[T]

// DrawHook はstate描画時のフック関数
// stateIndex: 現在描画したstateのインデックス
// stateCount: 現在のstateの総数
type DrawHook[T any] func(
	stateIndex int,
	stateCount int,
	world T,
	screen *ebiten.Image,
) error

// StateMachine はジェネリックな状態スタックを管理する
type StateMachine[T any] struct {
	states         []State[T]
	lastTransition Transition[T]
	AfterDrawHook  DrawHook[T]
}

// Init は新しいステートマシンを初期化する
func Init[T any](s State[T], world T) (StateMachine[T], error) {
	if err := s.OnStart(world); err != nil {
		return StateMachine[T]{}, err
	}
	return StateMachine[T]{
		states:         []State[T]{s},
		lastTransition: Transition[T]{Type: TransNone},
	}, nil
}

// Update はステートマシンを更新する
func (sm *StateMachine[T]) Update(world T) error {
	// ファクトリー関数からステートを作成
	states := sm.createStatesFromFunc()

	switch sm.lastTransition.Type {
	case TransNone:
		// 何もしない
	case TransPop:
		if err := sm.pop(world); err != nil {
			return err
		}
	case TransPush:
		if err := sm.push(world, states); err != nil {
			return err
		}
	case TransSwitch:
		if err := sm.switchState(world, states); err != nil {
			return err
		}
	case TransReplace:
		if err := sm.replace(world, states); err != nil {
			return err
		}
	case TransQuit:
		if err := sm.quit(world); err != nil {
			return err
		}
	}

	if len(sm.states) < 1 {
		return nil
	}

	// アクティブなステートを更新
	trans, err := sm.states[len(sm.states)-1].Update(world)
	if err != nil {
		return err
	}
	sm.lastTransition = trans
	return nil
}

// Draw は画面を描画する
func (sm *StateMachine[T]) Draw(world T, screen *ebiten.Image) error {
	stateCount := len(sm.states)
	for i, state := range sm.states {
		if err := state.Draw(world, screen); err != nil {
			return err
		}

		// 各state描画後にフックを呼び出し
		if sm.AfterDrawHook != nil {
			if err := sm.AfterDrawHook(i, stateCount, world, screen); err != nil {
				return err
			}
		}
	}

	return nil
}

// createStatesFromFunc はファクトリー関数からステートを作成する
func (sm *StateMachine[T]) createStatesFromFunc() []State[T] {
	if len(sm.lastTransition.NewStateFuncs) == 0 {
		return []State[T]{}
	}

	states := make([]State[T], len(sm.lastTransition.NewStateFuncs))
	for i, factory := range sm.lastTransition.NewStateFuncs {
		states[i] = factory()
	}
	return states
}

// pop はアクティブなステートを削除して次のステートを再開する
func (sm *StateMachine[T]) pop(world T) error {
	if len(sm.states) == 0 {
		return nil
	}

	currentState := sm.states[len(sm.states)-1]
	if err := currentState.OnStop(world); err != nil {
		return err
	}
	sm.states = sm.states[:len(sm.states)-1]

	if len(sm.states) > 0 {
		resumeState := sm.states[len(sm.states)-1]
		if err := resumeState.OnResume(world); err != nil {
			return err
		}
	}
	return nil
}

// push はアクティブなステートを一時停止して新しいステートをスタックに追加する
func (sm *StateMachine[T]) push(world T, newStates []State[T]) error {
	if len(newStates) == 0 {
		return nil
	}

	if len(sm.states) > 0 {
		currentState := sm.states[len(sm.states)-1]
		if err := currentState.OnPause(world); err != nil {
			return err
		}
	}

	for _, state := range newStates[:len(newStates)-1] {
		if err := state.OnStart(world); err != nil {
			return err
		}
		if err := state.OnPause(world); err != nil {
			return err
		}
	}

	activeState := newStates[len(newStates)-1]
	if err := activeState.OnStart(world); err != nil {
		return err
	}

	sm.states = append(sm.states, newStates...)
	return nil
}

// switchState はアクティブなステートを新しいものに置き換える
func (sm *StateMachine[T]) switchState(world T, newStates []State[T]) error {
	if len(newStates) != 1 {
		return nil
	}

	if len(sm.states) > 0 {
		currentState := sm.states[len(sm.states)-1]
		newState := newStates[0]

		if err := currentState.OnStop(world); err != nil {
			return err
		}
		if err := newState.OnStart(world); err != nil {
			return err
		}
		sm.states[len(sm.states)-1] = newState
	}
	return nil
}

// replace はすべてのステートを削除して新しいスタックを挿入する
func (sm *StateMachine[T]) replace(world T, newStates []State[T]) error {
	for len(sm.states) > 0 {
		currentState := sm.states[len(sm.states)-1]
		if err := currentState.OnStop(world); err != nil {
			return err
		}
		sm.states = sm.states[:len(sm.states)-1]
	}

	if len(newStates) > 0 {
		for _, state := range newStates[:len(newStates)-1] {
			if err := state.OnStart(world); err != nil {
				return err
			}
			if err := state.OnPause(world); err != nil {
				return err
			}
		}
		activeState := newStates[len(newStates)-1]
		if err := activeState.OnStart(world); err != nil {
			return err
		}
	}
	sm.states = newStates
	return nil
}

// quit はすべてのステートを削除して終了する
func (sm *StateMachine[T]) quit(world T) error {
	for len(sm.states) > 0 {
		currentState := sm.states[len(sm.states)-1]
		if err := currentState.OnStop(world); err != nil {
			return err
		}
		sm.states = sm.states[:len(sm.states)-1]
	}
	return nil
}

// GetStates はステートスタックのコピーを返す（テスト用）
func (sm *StateMachine[T]) GetStates() []State[T] {
	states := make([]State[T], len(sm.states))
	copy(states, sm.states)
	return states
}

// GetCurrentState は現在アクティブなステートを返す（テスト用）
func (sm *StateMachine[T]) GetCurrentState() State[T] {
	if len(sm.states) == 0 {
		return nil
	}
	return sm.states[len(sm.states)-1]
}

// GetStateCount はステートスタックの数を返す（テスト用）
func (sm *StateMachine[T]) GetStateCount() int {
	return len(sm.states)
}
