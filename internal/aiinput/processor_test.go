package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine_Hostile(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}

	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// 1ターン目：まだ待機継続
	rp.updateState(state, policy, false, 2)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "1ターン経過時は待機継続")

	// 3ターン目：待機時間終了で移動状態へ
	rp.updateState(state, policy, false, 3)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "待機時間終了で移動状態へ遷移")

	// プレイヤー発見で追跡状態へ
	rp.updateState(state, policy, true, 4)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "Hostileはプレイヤー発見で追跡状態へ遷移")
}

func TestStateMachine_Neutral(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore}

	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// プレイヤーを発見しても追跡しない
	rp.updateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "Neutralはプレイヤーを見ても追跡しない")
}

func TestStateMachine_Cowardly(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}

	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// プレイヤー発見で逃亡状態へ
	rp.updateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateFleeing, state.SubState, "Cowardlyはプレイヤー発見で逃亡状態へ遷移")
}

func TestStateMachine_Fleeing_Recovery(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}

	state := &gc.AIState{
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 3,
	}

	// 逃亡中にプレイヤーが見えている間は逃亡継続
	rp.updateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateFleeing, state.SubState, "プレイヤーが見えている間は逃亡継続")

	// プレイヤーを見失い、逃亡時間終了でデフォルトに復帰
	state.StartSubStateTurn = 1
	rp.updateState(state, policy, false, 5)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "プレイヤーを見失い逃亡時間終了で移動へ")
	assert.Equal(t, gc.CombatEvade, policy.CombatCurrent, "デフォルト態度に復帰")
}

func TestStateMachine_NilPolicy(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()

	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// AIPolicyがnilでも既存動作を維持する
	rp.updateState(state, nil, true, 2)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "AIPolicyなしはデフォルトで追跡")
}

func TestStateMachine_NeutralToHostile_StartChasing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore}

	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// Neutralはプレイヤーを見ても追跡しない
	rp.updateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "Neutralは追跡しない")

	// 被ダメージでCombatCurrentがCombatAttackに変化した（ReactToHostile相当）
	policy.CombatCurrent = gc.CombatAttack

	// 次のターンでプレイヤーを見たら追跡を開始する
	rp.updateState(state, policy, true, 3)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "Hostile化後はプレイヤー発見で追跡開始")
}

func TestStateMachine_CowardlyToFleeing_StartFleeing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}

	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	rp.updateState(state, policy, false, 2)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "プレイヤーが見えてなければまだ待機")

	rp.updateState(state, policy, true, 3)
	assert.Equal(t, gc.AIStateFleeing, state.SubState, "CombatEvadeはプレイヤー発見で逃亡開始")
}

func TestVisionSystem(t *testing.T) {
	t.Parallel()

	vs := NewVisionSystem()
	assert.NotNil(t, vs, "VisionSystemが作成できること")
}

func TestProcessor(t *testing.T) {
	t.Parallel()

	processor := NewProcessor()
	assert.NotNil(t, processor, "Processorが作成できること")
	assert.NotNil(t, processor.roamingPlanner)
	assert.NotNil(t, processor.squadPlanner)
}
