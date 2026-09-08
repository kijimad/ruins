package states

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errFailingState はFailingStateが注入するエラーを識別するセンチネル
var errFailingState = errors.New("failing state error")

// FailingState は各フックへ個別にエラーを注入できるテスト用state
type FailingState struct {
	failOnStart  bool
	failOnStop   bool
	failOnPause  bool
	failOnResume bool
	failUpdate   bool
	failDraw     bool

	onStartCalled  bool
	onStopCalled   bool
	onPauseCalled  bool
	onResumeCalled bool
}

func (fs *FailingState) OnStart(_ TestWorld) error {
	fs.onStartCalled = true
	if fs.failOnStart {
		return errFailingState
	}
	return nil
}

func (fs *FailingState) OnStop(_ TestWorld) error {
	fs.onStopCalled = true
	if fs.failOnStop {
		return errFailingState
	}
	return nil
}

func (fs *FailingState) OnPause(_ TestWorld) error {
	fs.onPauseCalled = true
	if fs.failOnPause {
		return errFailingState
	}
	return nil
}

func (fs *FailingState) OnResume(_ TestWorld) error {
	fs.onResumeCalled = true
	if fs.failOnResume {
		return errFailingState
	}
	return nil
}

func (fs *FailingState) Update(_ TestWorld) (Transition[TestWorld], error) {
	if fs.failUpdate {
		return Transition[TestWorld]{}, errFailingState
	}
	return Transition[TestWorld]{Type: TransNone}, nil
}

func (fs *FailingState) Draw(_ TestWorld, _ *ebiten.Image) error {
	if fs.failDraw {
		return errFailingState
	}
	return nil
}

func TestInit_OnStartがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	state := &FailingState{failOnStart: true}

	_, err := Init(state, world)

	require.ErrorIs(t, err, errFailingState)
}

func TestStateMachineUpdate_ファクトリー関数がエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	sm, err := Init(&TestState{name: "Base"}, world)
	require.NoError(t, err)

	sm.lastTransition = Transition[TestWorld]{
		Type: TransPush,
		NewStateFuncs: []StateFactory[TestWorld]{
			func() (State[TestWorld], error) { return nil, errFailingState },
		},
	}

	err = sm.Update(world)

	require.ErrorIs(t, err, errFailingState)
}

func TestStateMachineUpdate_アクティブstateのUpdateがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	failing := &FailingState{failUpdate: true}
	sm, err := Init(failing, world)
	require.NoError(t, err)

	err = sm.Update(world)

	require.ErrorIs(t, err, errFailingState)
}

func TestStateMachineDraw_いずれかのstateのDrawがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	ok := &TestState{name: "OK"}
	failing := &FailingState{failDraw: true}
	sm, err := Init(ok, world)
	require.NoError(t, err)
	err = sm.PushState(world, failing)
	require.NoError(t, err)

	err = sm.Draw(world, nil)

	require.ErrorIs(t, err, errFailingState)
}

func TestStateMachineUpdate_Pop遷移でエラーが伝播する(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	failing := &FailingState{failOnStop: true}
	sm := StateMachine[TestWorld]{
		states:         []State[TestWorld]{failing},
		lastTransition: Transition[TestWorld]{Type: TransPop},
	}

	err := sm.Update(world)

	require.ErrorIs(t, err, errFailingState)
}

func TestStateMachineUpdate_Push遷移でエラーが伝播する(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	base := &TestState{name: "Base"}
	sm, err := Init(base, world)
	require.NoError(t, err)
	sm.lastTransition = Transition[TestWorld]{
		Type: TransPush,
		NewStateFuncs: []StateFactory[TestWorld]{
			func() (State[TestWorld], error) { return &FailingState{failOnStart: true}, nil },
		},
	}

	err = sm.Update(world)

	require.ErrorIs(t, err, errFailingState)
}

func TestStateMachineUpdate_Switch遷移でエラーが伝播する(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	base := &TestState{name: "Base"}
	sm, err := Init(base, world)
	require.NoError(t, err)
	sm.lastTransition = Transition[TestWorld]{
		Type: TransSwitch,
		NewStateFuncs: []StateFactory[TestWorld]{
			func() (State[TestWorld], error) { return &FailingState{failOnStart: true}, nil },
		},
	}

	err = sm.Update(world)

	require.ErrorIs(t, err, errFailingState)
}

func TestStateMachineUpdate_Replace遷移でエラーが伝播する(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	base := &TestState{name: "Base"}
	sm, err := Init(base, world)
	require.NoError(t, err)
	sm.lastTransition = Transition[TestWorld]{
		Type: TransReplace,
		NewStateFuncs: []StateFactory[TestWorld]{
			func() (State[TestWorld], error) { return &FailingState{failOnStart: true}, nil },
		},
	}

	err = sm.Update(world)

	require.ErrorIs(t, err, errFailingState)
}

func TestStateMachineUpdate_Quit遷移でエラーが伝播する(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	failing := &FailingState{failOnStop: true}
	sm := StateMachine[TestWorld]{
		states:         []State[TestWorld]{failing},
		lastTransition: Transition[TestWorld]{Type: TransQuit},
	}

	err := sm.Update(world)

	require.ErrorIs(t, err, errFailingState)
}

func TestPop_空スタックでは何もしない(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	sm := StateMachine[TestWorld]{}

	err := sm.pop(world)

	require.NoError(t, err)
	assert.Equal(t, 0, sm.GetStateCount())
}

func TestPop_対象stateのOnStopがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	failing := &FailingState{failOnStop: true}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{failing}}

	err := sm.pop(world)

	require.ErrorIs(t, err, errFailingState)
	assert.Equal(t, 1, sm.GetStateCount(), "OnStopが失敗した場合はスタックから取り除かれない")
}

func TestPop_再開stateのOnResumeがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	base := &FailingState{failOnResume: true}
	top := &TestState{name: "Top"}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{base, top}}

	err := sm.pop(world)

	require.ErrorIs(t, err, errFailingState)
	assert.Equal(t, 1, sm.GetStateCount(), "OnStop済みのtopは取り除かれたまま")
	assert.True(t, top.onStopCalled)
}

func TestPush_現在stateのOnPauseがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	base := &FailingState{failOnPause: true}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{base}}
	newState := &TestState{name: "New"}

	err := sm.push(world, []State[TestWorld]{newState})

	require.ErrorIs(t, err, errFailingState)
	assert.Equal(t, 1, sm.GetStateCount(), "失敗時は新しいstateを追加しない")
	assert.False(t, newState.onStartCalled)
}

func TestPush_中間stateのOnStartがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	base := &TestState{name: "Base"}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{base}}
	middle := &FailingState{failOnStart: true}
	top := &TestState{name: "Top"}

	err := sm.push(world, []State[TestWorld]{middle, top})

	require.ErrorIs(t, err, errFailingState)
	assert.Equal(t, 1, sm.GetStateCount())
	assert.False(t, top.onStartCalled, "先行するstateが失敗したら後続は処理されない")
}

func TestPush_中間stateのOnPauseがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	base := &TestState{name: "Base"}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{base}}
	middle := &FailingState{failOnPause: true}
	top := &TestState{name: "Top"}

	err := sm.push(world, []State[TestWorld]{middle, top})

	require.ErrorIs(t, err, errFailingState)
	assert.Equal(t, 1, sm.GetStateCount())
	assert.True(t, middle.onStartCalled)
	assert.False(t, top.onStartCalled)
}

func TestPush_対象stateのOnStartがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	base := &TestState{name: "Base"}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{base}}
	top := &FailingState{failOnStart: true}

	err := sm.push(world, []State[TestWorld]{top})

	require.ErrorIs(t, err, errFailingState)
	assert.Equal(t, 1, sm.GetStateCount(), "失敗時はスタックに追加しない")
}

func TestSwitchState_newStatesの数が1でない場合は何もしない(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}

	t.Run("0個の場合", func(t *testing.T) {
		t.Parallel()
		base := &TestState{name: "Base"}
		sm := StateMachine[TestWorld]{states: []State[TestWorld]{base}}

		err := sm.switchState(world, []State[TestWorld]{})

		require.NoError(t, err)
		assert.False(t, base.onStopCalled)
		assert.Equal(t, 1, sm.GetStateCount())
	})

	t.Run("2個の場合", func(t *testing.T) {
		t.Parallel()
		base := &TestState{name: "Base"}
		sm := StateMachine[TestWorld]{states: []State[TestWorld]{base}}
		a := &TestState{name: "A"}
		b := &TestState{name: "B"}

		err := sm.switchState(world, []State[TestWorld]{a, b})

		require.NoError(t, err)
		assert.False(t, base.onStopCalled)
		assert.Equal(t, 1, sm.GetStateCount())
	})
}

func TestSwitchState_現在stateのOnStopがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	current := &FailingState{failOnStop: true}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{current}}
	newState := &TestState{name: "New"}

	err := sm.switchState(world, []State[TestWorld]{newState})

	require.ErrorIs(t, err, errFailingState)
	assert.False(t, newState.onStartCalled, "OnStopが失敗したら新stateは開始されない")
	assert.Equal(t, 1, sm.GetStateCount())
}

func TestSwitchState_新しいstateのOnStartがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	current := &TestState{name: "Current"}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{current}}
	newState := &FailingState{failOnStart: true}

	err := sm.switchState(world, []State[TestWorld]{newState})

	require.ErrorIs(t, err, errFailingState)
	// OnStopは成功するが新stateのOnStart失敗で置き換えは行われず、停止済みのcurrentが残る
	assert.True(t, current.onStopCalled)
	cs, ok := sm.GetCurrentState().(*TestState)
	require.True(t, ok, "置き換え失敗時はスタックの内容自体は変わらない")
	assert.Same(t, current, cs)
}

func TestReplace_現在stateのOnStopがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	current := &FailingState{failOnStop: true}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{current}}
	newState := &TestState{name: "New"}

	err := sm.replace(world, []State[TestWorld]{newState})

	require.ErrorIs(t, err, errFailingState)
	assert.Equal(t, 1, sm.GetStateCount(), "OnStopが失敗した場合は取り除かれない")
	assert.False(t, newState.onStartCalled)
}

func TestReplace_中間stateのOnStartがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	old := &TestState{name: "Old"}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{old}}
	middle := &FailingState{failOnStart: true}
	top := &TestState{name: "Top"}

	err := sm.replace(world, []State[TestWorld]{middle, top})

	require.ErrorIs(t, err, errFailingState)
	assert.True(t, old.onStopCalled)
	assert.False(t, top.onStartCalled)
	assert.Equal(t, 0, sm.GetStateCount(), "旧stateは除去済みだが新stateは設定されない")
}

func TestReplace_中間stateのOnPauseがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	old := &TestState{name: "Old"}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{old}}
	middle := &FailingState{failOnPause: true}
	top := &TestState{name: "Top"}

	err := sm.replace(world, []State[TestWorld]{middle, top})

	require.ErrorIs(t, err, errFailingState)
	assert.True(t, middle.onStartCalled)
	assert.False(t, top.onStartCalled)
	assert.Equal(t, 0, sm.GetStateCount())
}

func TestReplace_対象stateのOnStartがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	old := &TestState{name: "Old"}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{old}}
	top := &FailingState{failOnStart: true}

	err := sm.replace(world, []State[TestWorld]{top})

	require.ErrorIs(t, err, errFailingState)
	assert.True(t, old.onStopCalled)
	assert.Equal(t, 0, sm.GetStateCount())
}

func TestQuit_OnStopがエラーを返す場合(t *testing.T) {
	t.Parallel()
	world := TestWorld{Name: "TestWorld"}
	bottom := &TestState{name: "Bottom"}
	top := &FailingState{failOnStop: true}
	sm := StateMachine[TestWorld]{states: []State[TestWorld]{bottom, top}}

	err := sm.quit(world)

	require.ErrorIs(t, err, errFailingState)
	assert.Equal(t, 2, sm.GetStateCount(), "OnStopが失敗したら取り除かれない")
}
