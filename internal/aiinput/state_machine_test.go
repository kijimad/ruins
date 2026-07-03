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

	rp := newRoamingPlanner()
	state := &gc.AIState{
		SubState:              gc.AIStateSubState("INVALID"),
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, nil, false, 10)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "不明な状態は待機に初期化される")
}

func TestUpdateState_ChasingLostPlayer(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}

	state := &gc.AIState{
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	// プレイヤーを見失って3ターン未満は追跡継続
	rp.updateState(state, policy, false, 3)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "3ターン未満は追跡継続")

	// 3ターン以上見失うと移動状態へ
	rp.updateState(state, policy, false, 5)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "3ターン以上見失うと移動状態へ")
}

func TestUpdateState_ChasingPlayerVisible_ResetsTurn(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}

	state := &gc.AIState{
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	rp.updateState(state, policy, true, 5)
	assert.Equal(t, gc.AIStateChasing, state.SubState)
	assert.Equal(t, 5, state.StartSubStateTurn, "プレイヤー視認中はターンリセット")
}

func TestUpdateState_WaitingToDriving(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 3,
	}

	rp.updateState(state, nil, false, 2)
	assert.Equal(t, gc.AIStateWaiting, state.SubState)

	rp.updateState(state, nil, false, 3)
	assert.Equal(t, gc.AIStateDriving, state.SubState)
}

func TestUpdateState_DrivingToWaiting(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, nil, false, 4)
	assert.Equal(t, gc.AIStateDriving, state.SubState)

	rp.updateState(state, nil, false, 5)
	assert.Equal(t, gc.AIStateWaiting, state.SubState)
}

func TestUpdateState_WaitingToChasing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateChasing, state.SubState)
}

func TestUpdateState_WaitingToFleeing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateFleeing, state.SubState)
}

func TestUpdateState_WaitingNeutralIgnoresPlayer(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore}
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateWaiting, state.SubState)
}

func TestUpdateState_DrivingToChasing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}
	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateChasing, state.SubState)
}

func TestUpdateState_DrivingToFleeing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateFleeing, state.SubState)
}

func TestUpdateState_FleeingPlayerVisible_ResetsTurn(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
	state := &gc.AIState{
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, policy, true, 3)
	assert.Equal(t, gc.AIStateFleeing, state.SubState)
	assert.Equal(t, 3, state.StartSubStateTurn)
}

func TestUpdateState_FleeingToDriving(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
	state := &gc.AIState{
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, policy, false, 5)
	assert.Equal(t, gc.AIStateDriving, state.SubState)
	assert.Equal(t, gc.CombatEvade, policy.CombatCurrent)
}

// TestCombatIgnorePermanentHostile はCombatIgnore NPCが被ダメージで永続的に敵対化することを検証する。
// 一度殴られたNPCは見失っても再発見時に追跡を再開する
func TestCombatIgnorePermanentHostile(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore}
	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	// 1. 初期状態: CombatIgnoreなのでプレイヤーを見ても無視する
	rp.updateState(state, policy, true, 1)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "CombatIgnoreはプレイヤーを無視する")

	// 2. 被ダメージ: ReactToHostileでCombatAttackに永続的に変化する
	policy.ReactToHostile()
	assert.Equal(t, gc.CombatAttack, policy.CombatCurrent)

	// 3. プレイヤーを発見して追跡を開始する
	rp.updateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "CombatAttackに変化したので追跡する")

	// 4. プレイヤーを見失い、3ターン以上経過して追跡を終了する
	rp.updateState(state, policy, false, 6)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "見失ってDrivingに遷移する")
	assert.Equal(t, gc.CombatAttack, policy.CombatCurrent, "敵対状態は維持される")

	// 5. Driving終了後にWaitingへ
	turnAfterDriving := state.StartSubStateTurn + state.DurationSubStateTurns
	rp.updateState(state, policy, false, turnAfterDriving)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "Driving終了でWaitingに戻る")

	// 6. 再発見時にまた追跡する
	rp.updateState(state, policy, true, turnAfterDriving+1)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "敵対が永続しているので再追跡する")
}

func TestUpdateState_ChasingTimeout(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	state := &gc.AIState{
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 10,
	}

	rp.updateState(state, nil, true, 10)
	assert.Equal(t, gc.AIStateWaiting, state.SubState)
}
