package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine_Hostile(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}

	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// 1ターン目：まだ待機継続
	sm.UpdateState(state, policy, false, 2)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "1ターン経過時は待機継続")

	// 3ターン目：待機時間終了で移動状態へ
	sm.UpdateState(state, policy, false, 3)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "待機時間終了で移動状態へ遷移")

	// プレイヤー発見で追跡状態へ
	sm.UpdateState(state, policy, true, 4)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "Hostileはプレイヤー発見で追跡状態へ遷移")
}

func TestStateMachine_Neutral(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore}

	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// プレイヤーを発見しても追跡しない
	sm.UpdateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "Neutralはプレイヤーを見ても追跡しない")
}

func TestStateMachine_Cowardly(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}

	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// プレイヤー発見で逃亡状態へ
	sm.UpdateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateFleeing, state.SubState, "Cowardlyはプレイヤー発見で逃亡状態へ遷移")
}

func TestStateMachine_Fleeing_Recovery(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}

	state := &gc.AIState{
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 3,
	}

	// 逃亡中にプレイヤーが見えている間は逃亡継続
	sm.UpdateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateFleeing, state.SubState, "プレイヤーが見えている間は逃亡継続")

	// プレイヤーを見失い、逃亡時間終了でデフォルトに復帰
	state.StartSubStateTurn = 1
	sm.UpdateState(state, policy, false, 5)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "プレイヤーを見失い逃亡時間終了で移動へ")
	assert.Equal(t, gc.CombatEvade, policy.CombatCurrent, "デフォルト態度に復帰")
}

func TestStateMachine_NilPolicy(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()

	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// AIPolicyがnilでも既存動作を維持する
	sm.UpdateState(state, nil, true, 2)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "AIPolicyなしはデフォルトで追跡")
}

func TestStateMachine_NeutralToHostile_StartChasing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore}

	state := &gc.AIState{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// Neutralはプレイヤーを見ても追跡しない
	sm.UpdateState(state, policy, true, 2)
	assert.Equal(t, gc.AIStateDriving, state.SubState, "Neutralは追跡しない")

	// 被ダメージでCombatCurrentがCombatAttackに変化した（ReactToHostile相当）
	policy.CombatCurrent = gc.CombatAttack

	// 次のターンでプレイヤーを見たら追跡を開始する
	sm.UpdateState(state, policy, true, 3)
	assert.Equal(t, gc.AIStateChasing, state.SubState, "Hostile化後はプレイヤー発見で追跡開始")
}

func TestStateMachine_CowardlyToFleeing_StartFleeing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	policy := &gc.AIPolicy{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}

	state := &gc.AIState{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// CombatEvadeはプレイヤーを見ていなければ待機継続
	// プレイヤーを見ていなくても逃亡状態へ遷移しない
	sm.UpdateState(state, policy, false, 2)
	assert.Equal(t, gc.AIStateWaiting, state.SubState, "プレイヤーが見えてなければまだ待機")

	// プレイヤーが見える場合は逃亡開始
	sm.UpdateState(state, policy, true, 3)
	assert.Equal(t, gc.AIStateFleeing, state.SubState, "CombatEvadeはプレイヤー発見で逃亡開始")
}

func TestVisionSystem(t *testing.T) {
	t.Parallel()

	// VisionSystemのテストは統合テストなので、ここでは基本的な動作のみ
	vs := NewVisionSystem()
	assert.NotNil(t, vs, "VisionSystemが作成できること")

	t.Logf("VisionSystemテスト完了")
}

func TestActionPlanner(t *testing.T) {
	t.Parallel()

	// ActionPlannerのテストも統合テストなので、ここでは基本的な動作のみ
	ap := NewActionPlanner()
	assert.NotNil(t, ap, "ActionPlannerが作成できること")

	t.Logf("ActionPlannerテスト完了")
}

func TestProcessor(t *testing.T) {
	t.Parallel()

	// Processorの基本作成テスト
	processor := NewProcessor()
	assert.NotNil(t, processor, "Processorが作成できること")

	t.Logf("Processorテスト完了")
}
