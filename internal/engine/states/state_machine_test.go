package states

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorld はテスト用のゲーム世界
type TestWorld struct {
	Name string
}

// TestState はテスト用の状態実装
type TestState struct {
	name           string
	onStartCalled  bool
	onStopCalled   bool
	onPauseCalled  bool
	onResumeCalled bool
	updateCalled   bool
	drawCalled     bool
}

func (ts *TestState) String() string {
	return ts.name
}

func (ts *TestState) OnStart(_ TestWorld) error {
	ts.onStartCalled = true
	return nil
}

func (ts *TestState) OnStop(_ TestWorld) error {
	ts.onStopCalled = true
	return nil
}

func (ts *TestState) OnPause(_ TestWorld) error {
	ts.onPauseCalled = true
	return nil
}

func (ts *TestState) OnResume(_ TestWorld) error {
	ts.onResumeCalled = true
	return nil
}

func (ts *TestState) Update(_ TestWorld) (Transition[TestWorld], error) {
	ts.updateCalled = true
	return Transition[TestWorld]{Type: TransNone}, nil
}

func (ts *TestState) Draw(_ TestWorld, _ *ebiten.Image) error {
	ts.drawCalled = true
	return nil
}

// TestGetStatesMethods はGetStatesメソッド群のテスト
func TestGetStatesMethods(t *testing.T) {
	t.Parallel()
	t.Run("初期状態での動作確認", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		initialState := &TestState{name: "InitialState"}

		// StateMachineの初期化
		stateMachine, err := Init(initialState, world)
		require.NoError(t, err)

		// GetStatesのテスト
		states := stateMachine.GetStates()
		assert.Len(t, states, 1, "初期状態の数が正しくない")
		assert.Equal(t, "InitialState", states[0].(*TestState).name, "初期状態の名前が正しくない")

		// GetCurrentStateのテスト
		currentState := stateMachine.GetCurrentState()
		assert.NotNil(t, currentState, "現在の状態がnil")
		assert.Equal(t, "InitialState", currentState.(*TestState).name, "現在の状態の名前が正しくない")

		// GetStateCountのテスト
		stateCount := stateMachine.GetStateCount()
		assert.Equal(t, 1, stateCount, "状態数が正しくない")

		// OnStartが呼ばれていることを確認
		assert.True(t, initialState.onStartCalled, "OnStartが呼ばれていない")
	})

	t.Run("状態の不変性確認", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		initialState := &TestState{name: "InitialState"}
		stateMachine, err := Init(initialState, world)
		require.NoError(t, err)

		// GetStatesで取得したスライスを変更しても元のスタックに影響しないことを確認
		states := stateMachine.GetStates()
		originalLength := len(states)

		// 取得したスライスを変更
		_ = append(states, &TestState{name: "ModifiedState"})

		// 元のスタックは変更されていないことを確認
		newStates := stateMachine.GetStates()
		assert.Len(t, newStates, originalLength, "元の状態スタックが変更されている")
		assert.Equal(t, "InitialState", newStates[0].(*TestState).name, "元の状態が変更されている")
	})

	t.Run("空の状態スタックでの動作", func(t *testing.T) {
		t.Parallel()
		// 空のStateMachineを作成（実際のゲームでは発生しないが、テスト用）
		stateMachine := StateMachine[TestWorld]{}

		// GetStatesのテスト
		states := stateMachine.GetStates()
		assert.Empty(t, states, "空のスタックの状態数が正しくない")

		// GetCurrentStateのテスト
		currentState := stateMachine.GetCurrentState()
		assert.Nil(t, currentState, "空のスタックの現在状態がnilでない")

		// GetStateCountのテスト
		stateCount := stateMachine.GetStateCount()
		assert.Equal(t, 0, stateCount, "空のスタックの状態数が正しくない")
	})
}

// TestPushState はPushStateメソッドのテスト
func TestPushState(t *testing.T) {
	t.Parallel()

	t.Run("1つのstateをpushする", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		base := &TestState{name: "Base"}
		sm, err := Init(base, world)
		require.NoError(t, err)

		pushed := &TestState{name: "Pushed"}
		err = sm.PushState(world, pushed)
		require.NoError(t, err)

		assert.Equal(t, 2, sm.GetStateCount())
		assert.Equal(t, "Pushed", sm.GetCurrentState().(*TestState).name)
		assert.True(t, base.onPauseCalled)
		assert.True(t, pushed.onStartCalled)
	})

	t.Run("複数のstateを一度にpushする", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		base := &TestState{name: "Base"}
		sm, err := Init(base, world)
		require.NoError(t, err)

		middle := &TestState{name: "Middle"}
		top := &TestState{name: "Top"}
		err = sm.PushState(world, middle, top)
		require.NoError(t, err)

		assert.Equal(t, 3, sm.GetStateCount())
		assert.Equal(t, "Top", sm.GetCurrentState().(*TestState).name)

		// 中間stateはStart後にPauseされる
		assert.True(t, middle.onStartCalled)
		assert.True(t, middle.onPauseCalled)
		// 最上位stateはStartのみ
		assert.True(t, top.onStartCalled)
		assert.False(t, top.onPauseCalled)
	})

	t.Run("空のstateリストでpushしてもエラーにならない", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		base := &TestState{name: "Base"}
		sm, err := Init(base, world)
		require.NoError(t, err)

		err = sm.PushState(world)
		require.NoError(t, err)
		assert.Equal(t, 1, sm.GetStateCount())
	})
}

// TestPushStateAndDraw はPushState後にDrawが全stateで呼ばれることを検証する
func TestPushStateAndDraw(t *testing.T) {
	t.Parallel()

	t.Run("PushState後にDrawが全stateで呼ばれる", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		base := &TestState{name: "Base"}
		sm, err := Init(base, world)
		require.NoError(t, err)

		// ベースstateのUpdateを実行してシステムを初期化する
		err = sm.Update(world)
		require.NoError(t, err)
		assert.True(t, base.updateCalled)

		// メニューstateをpushする
		menu := &TestState{name: "Menu"}
		err = sm.PushState(world, menu)
		require.NoError(t, err)

		// drawCalledをリセットして検証する
		base.drawCalled = false
		menu.drawCalled = false

		err = sm.Draw(world, nil)
		require.NoError(t, err)

		assert.True(t, base.drawCalled, "ベースstateのDrawが呼ばれていない")
		assert.True(t, menu.drawCalled, "メニューstateのDrawが呼ばれていない")
	})

	t.Run("PushState後のstateスタック順序が正しい", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		base := &TestState{name: "Base"}
		sm, err := Init(base, world)
		require.NoError(t, err)

		err = sm.Update(world)
		require.NoError(t, err)

		menu := &TestState{name: "Menu"}
		err = sm.PushState(world, menu)
		require.NoError(t, err)

		states := sm.GetStates()
		assert.Len(t, states, 2)
		assert.Equal(t, "Base", states[0].(*TestState).name, "スタックの底がベースstateであること")
		assert.Equal(t, "Menu", states[1].(*TestState).name, "スタックの上がメニューstateであること")
	})
}

// TestStateMachineTransitions は状態遷移のテスト
func TestStateMachineTransitions(t *testing.T) {
	t.Parallel()
	t.Run("Push遷移のテスト", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		initialState := &TestState{name: "InitialState"}
		stateMachine, err := Init(initialState, world)
		require.NoError(t, err)

		// Push遷移を実行
		newState := &TestState{name: "PushedState"}
		stateMachine.lastTransition = Transition[TestWorld]{
			Type:          TransPush,
			NewStateFuncs: []StateFactory[TestWorld]{func() (State[TestWorld], error) { return newState, nil }},
		}
		err = stateMachine.Update(world)
		require.NoError(t, err, "Push遷移でエラーが発生")

		// 状態数の確認
		assert.Equal(t, 2, stateMachine.GetStateCount(), "Push後の状態数が正しくない")

		// 現在の状態の確認
		currentState := stateMachine.GetCurrentState()
		assert.Equal(t, "PushedState", currentState.(*TestState).name, "Push後の現在状態が正しくない")

		// 状態スタックの確認
		states := stateMachine.GetStates()
		assert.Len(t, states, 2, "Push後の状態スタック数が正しくない")
		assert.Equal(t, "InitialState", states[0].(*TestState).name, "Push後の最初の状態が正しくない")
		assert.Equal(t, "PushedState", states[1].(*TestState).name, "Push後の最後の状態が正しくない")

		// 初期状態がPauseされていることを確認
		assert.True(t, initialState.onPauseCalled, "初期状態のOnPauseが呼ばれていない")
		// 新しい状態がStartされていることを確認
		assert.True(t, newState.onStartCalled, "新しい状態のOnStartが呼ばれていない")
	})

	t.Run("Pop遷移のテスト", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		initialState := &TestState{name: "InitialState"}
		stateMachine, err := Init(initialState, world)
		require.NoError(t, err)

		// まずPushして2つの状態にする
		pushedState := &TestState{name: "PushedState"}
		stateMachine.lastTransition = Transition[TestWorld]{
			Type:          TransPush,
			NewStateFuncs: []StateFactory[TestWorld]{func() (State[TestWorld], error) { return pushedState, nil }},
		}
		err = stateMachine.Update(world)
		require.NoError(t, err, "Push遷移でエラーが発生")

		// Pop遷移を実行
		stateMachine.lastTransition = Transition[TestWorld]{Type: TransPop}
		err = stateMachine.Update(world)
		require.NoError(t, err, "Pop遷移でエラーが発生")

		// 状態数の確認
		assert.Equal(t, 1, stateMachine.GetStateCount(), "Pop後の状態数が正しくない")

		// 現在の状態の確認
		currentState := stateMachine.GetCurrentState()
		assert.Equal(t, "InitialState", currentState.(*TestState).name, "Pop後の現在状態が正しくない")

		// Popされた状態のOnStopが呼ばれていることを確認
		assert.True(t, pushedState.onStopCalled, "Popされた状態のOnStopが呼ばれていない")
		// 再開された状態のOnResumeが呼ばれていることを確認
		assert.True(t, initialState.onResumeCalled, "再開された状態のOnResumeが呼ばれていない")
	})

	t.Run("Switch遷移のテスト", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		initialState := &TestState{name: "InitialState"}
		stateMachine, err := Init(initialState, world)
		require.NoError(t, err)

		// Switch遷移を実行
		newState := &TestState{name: "SwitchedState"}
		stateMachine.lastTransition = Transition[TestWorld]{
			Type:          TransSwitch,
			NewStateFuncs: []StateFactory[TestWorld]{func() (State[TestWorld], error) { return newState, nil }},
		}
		err = stateMachine.Update(world)
		require.NoError(t, err, "Switch遷移でエラーが発生")

		// 状態数の確認（変わらず1つ）
		assert.Equal(t, 1, stateMachine.GetStateCount(), "Switch後の状態数が正しくない")

		// 現在の状態の確認
		currentState := stateMachine.GetCurrentState()
		assert.Equal(t, "SwitchedState", currentState.(*TestState).name, "Switch後の現在状態が正しくない")

		// 初期状態のOnStopが呼ばれていることを確認
		assert.True(t, initialState.onStopCalled, "初期状態のOnStopが呼ばれていない")
		// 新しい状態のOnStartが呼ばれていることを確認
		assert.True(t, newState.onStartCalled, "新しい状態のOnStartが呼ばれていない")
	})

	t.Run("Replace遷移のテスト", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		state1 := &TestState{name: "State1"}
		sm, err := Init(state1, world)
		require.NoError(t, err)

		// 2つ目をpush
		state2 := &TestState{name: "State2"}
		sm.lastTransition = Transition[TestWorld]{
			Type:          TransPush,
			NewStateFuncs: []StateFactory[TestWorld]{func() (State[TestWorld], error) { return state2, nil }},
		}
		err = sm.Update(world)
		require.NoError(t, err)
		assert.Equal(t, 2, sm.GetStateCount())

		// Replace遷移で全て置き換え
		newState := &TestState{name: "Replaced"}
		sm.lastTransition = Transition[TestWorld]{
			Type:          TransReplace,
			NewStateFuncs: []StateFactory[TestWorld]{func() (State[TestWorld], error) { return newState, nil }},
		}
		err = sm.Update(world)
		require.NoError(t, err)

		assert.Equal(t, 1, sm.GetStateCount())
		assert.Equal(t, "Replaced", sm.GetCurrentState().(*TestState).name)
		assert.True(t, state1.onStopCalled)
		assert.True(t, state2.onStopCalled)
		assert.True(t, newState.onStartCalled)
	})

	t.Run("Quit遷移のテスト", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		state1 := &TestState{name: "State1"}
		sm, err := Init(state1, world)
		require.NoError(t, err)

		state2 := &TestState{name: "State2"}
		sm.lastTransition = Transition[TestWorld]{
			Type:          TransPush,
			NewStateFuncs: []StateFactory[TestWorld]{func() (State[TestWorld], error) { return state2, nil }},
		}
		err = sm.Update(world)
		require.NoError(t, err)

		// Quit遷移で全状態を削除
		sm.lastTransition = Transition[TestWorld]{Type: TransQuit}
		err = sm.Update(world)
		require.NoError(t, err)

		assert.Equal(t, 0, sm.GetStateCount())
		assert.Nil(t, sm.GetCurrentState())
		assert.True(t, state1.onStopCalled)
		assert.True(t, state2.onStopCalled)
	})

	t.Run("Replace遷移で複数ステートを挿入", func(t *testing.T) {
		t.Parallel()
		world := TestWorld{Name: "TestWorld"}
		sm, err := Init(&TestState{name: "Old"}, world)
		require.NoError(t, err)

		bottom := &TestState{name: "Bottom"}
		top := &TestState{name: "Top"}
		sm.lastTransition = Transition[TestWorld]{
			Type: TransReplace,
			NewStateFuncs: []StateFactory[TestWorld]{
				func() (State[TestWorld], error) { return bottom, nil },
				func() (State[TestWorld], error) { return top, nil },
			},
		}
		err = sm.Update(world)
		require.NoError(t, err)

		assert.Equal(t, 2, sm.GetStateCount())
		assert.Equal(t, "Top", sm.GetCurrentState().(*TestState).name)
		assert.True(t, bottom.onStartCalled)
		assert.True(t, bottom.onPauseCalled)
		assert.True(t, top.onStartCalled)
		assert.False(t, top.onPauseCalled)
	})
}
