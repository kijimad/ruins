package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestShouldChase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy *gc.AIPolicy
		want   bool
	}{
		{"CombatAttackは追跡する", &gc.AIPolicy{CombatCurrent: gc.CombatAttack}, true},
		{"CombatIgnoreは追跡しない", &gc.AIPolicy{CombatCurrent: gc.CombatIgnore}, false},
		{"CombatEvadeは追跡しない", &gc.AIPolicy{CombatCurrent: gc.CombatEvade}, false},
		{"nilは追跡する", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shouldChase(tt.policy))
		})
	}
}

func TestShouldFlee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy *gc.AIPolicy
		want   bool
	}{
		{"CombatEvadeは逃亡する", &gc.AIPolicy{CombatCurrent: gc.CombatEvade}, true},
		{"CombatAttackは逃亡しない", &gc.AIPolicy{CombatCurrent: gc.CombatAttack}, false},
		{"CombatIgnoreは逃亡しない", &gc.AIPolicy{CombatCurrent: gc.CombatIgnore}, false},
		{"nilは逃亡しない", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shouldFlee(tt.policy))
		})
	}
}

func TestUpdateState_UnknownState(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	state := &gc.AIState{
		SubState:              gc.AIStateSubState("INVALID"),
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(state, nil, false, 10)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "不明な状態は待機に初期化される")
}

func TestUpdateState_ChasingLostPlayer(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}

	state := &gc.AIState{
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	// プレイヤーを見失って3ターン未満は追跡継続
	sm.UpdateState(state, policy, false, 3)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "3ターン未満は追跡継続")

	// 3ターン以上見失うと移動状態へ
	sm.UpdateState(state, policy, false, 5)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "3ターン以上見失うと移動状態へ")
}

func TestUpdateState_ChasingPlayerVisible_ResetsTurn(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}

	state := &gc.AIState{
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	// プレイヤーが見えている間はターンリセット
	sm.UpdateState(state, policy, true, 5)
	assert.Equal(t, gc.AIStateChasing, state.SubState)
	assert.Equal(t, 5, state.StartSubStateTurn, "プレイヤー視認中はターンリセット")
}

func TestUpdateState_WaitingToDriving(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 3,
	}

	// 待機時間未満は待機のまま
	sm.UpdateState(state, nil, false, 2)
	assert.Equal(t, gc.AIStateWaiting, state.SubState)

	// 待機時間経過で移動へ
	sm.UpdateState(state, nil, false, 3)
	assert.Equal(t, gc.AIStateDriving, state.SubState)
}

func TestUpdateState_DrivingToWaiting(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	// 移動時間未満は移動のまま
	sm.UpdateState(state, nil, false, 4)
	assert.Equal(t, gc.AIStateDriving, state.SubState)

	// 移動時間経過で待機へ
	sm.UpdateState(state, nil, false, 5)
	assert.Equal(t, gc.AIStateWaiting, state.SubState)
}

func TestUpdateState_WaitingToChasing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateChasing, state.SubState)
}

func TestUpdateState_WaitingToFleeing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateFleeing, state.SubState)
}

func TestUpdateState_WaitingNeutralIgnoresPlayer(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore}
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateWaiting, state.SubState)
}

func TestUpdateState_DrivingToChasing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}
	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateChasing, state.SubState)
}

func TestUpdateState_DrivingToFleeing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateFleeing, state.SubState)
}

func TestUpdateState_FleeingPlayerVisible_ResetsTurn(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
	state := &gc.AIState{
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(state, policy, true, 3)
	assert.Equal(t, gc.AIStateFleeing, state.SubState)
	assert.Equal(t, 3, state.StartSubStateTurn)
}

func TestUpdateState_FleeingToDriving(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
	state := &gc.AIState{
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(state, policy, false, 5)
	assert.Equal(t, gc.AIStateDriving, state.SubState)
	assert.Equal(t, gc.CombatEvade, policy.CombatCurrent)
}

// policy=nil は CombatAttack として扱われる。
// 追跡タイムアウトが経過するとプレイヤーが見えていても Waiting に戻る
func TestUpdateState_ChasingTimeout(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	state := &gc.AIState{
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 10,
	}

	sm.UpdateState(state, nil, true, 10)
	assert.Equal(t, gc.AIStateWaiting, state.SubState)
}
