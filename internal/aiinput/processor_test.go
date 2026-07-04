package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine_Hostile(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner(testRNG)

	ai := &gc.AI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// 1ターン目：まだ待機継続
	rp.updateState(ai, false, 2)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState, "1ターン経過時は待機継続")

	// 3ターン目：待機時間終了で移動状態へ
	rp.updateState(ai, false, 3)
	assert.Equal(t, gc.AIStateDriving, ai.SubState, "待機時間終了で移動状態へ遷移")

	// プレイヤー発見で追跡状態へ
	rp.updateState(ai, true, 4)
	assert.Equal(t, gc.AIStateChasing, ai.SubState, "Hostileはプレイヤー発見で追跡状態へ遷移")
}

func TestStateMachine_Neutral(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner(testRNG)

	ai := &gc.AI{
		CombatDefault:         gc.CombatIgnore,
		CombatCurrent:         gc.CombatIgnore,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// プレイヤーを発見しても追跡しない
	rp.updateState(ai, true, 2)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState, "Neutralはプレイヤーを見ても追跡しない")
}

func TestStateMachine_Cowardly(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner(testRNG)

	ai := &gc.AI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// プレイヤー発見で逃亡状態へ
	rp.updateState(ai, true, 2)
	assert.Equal(t, gc.AIStateFleeing, ai.SubState, "Cowardlyはプレイヤー発見で逃亡状態へ遷移")
}

func TestStateMachine_Fleeing_Recovery(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner(testRNG)

	ai := &gc.AI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 3,
	}

	// 逃亡中にプレイヤーが見えている間は逃亡継続
	rp.updateState(ai, true, 2)
	assert.Equal(t, gc.AIStateFleeing, ai.SubState, "プレイヤーが見えている間は逃亡継続")

	// プレイヤーを見失い、逃亡時間終了でデフォルトに復帰
	ai.StartSubStateTurn = 1
	rp.updateState(ai, false, 5)
	assert.Equal(t, gc.AIStateDriving, ai.SubState, "プレイヤーを見失い逃亡時間終了で移動へ")
	assert.Equal(t, gc.CombatEvade, ai.CombatCurrent, "デフォルト態度に復帰")
}

func TestStateMachine_NeutralToHostile_StartChasing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner(testRNG)

	ai := &gc.AI{
		CombatDefault:         gc.CombatIgnore,
		CombatCurrent:         gc.CombatIgnore,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// Neutralはプレイヤーを見ても追跡しない
	rp.updateState(ai, true, 2)
	assert.Equal(t, gc.AIStateDriving, ai.SubState, "Neutralは追跡しない")

	// 被ダメージでCombatCurrentがCombatAttackに変化した（ReactToHostile相当）
	ai.CombatCurrent = gc.CombatAttack

	// 次のターンでプレイヤーを見たら追跡を開始する
	rp.updateState(ai, true, 3)
	assert.Equal(t, gc.AIStateChasing, ai.SubState, "Hostile化後はプレイヤー発見で追跡開始")
}

func TestStateMachine_CowardlyToFleeing_StartFleeing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner(testRNG)

	ai := &gc.AI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, false, 2)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState, "プレイヤーが見えてなければまだ待機")

	rp.updateState(ai, true, 3)
	assert.Equal(t, gc.AIStateFleeing, ai.SubState, "CombatEvadeはプレイヤー発見で逃亡開始")
}

func TestVisionSystem(t *testing.T) {
	t.Parallel()

	vs := NewVisionSystem()
	assert.NotNil(t, vs, "VisionSystemが作成できること")
}

func TestProcessor(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(testRNG)
	assert.NotNil(t, processor, "Processorが作成できること")
	assert.NotNil(t, processor.planners[gc.PlannerRoaming])
	assert.NotNil(t, processor.planners[gc.PlannerSquad])
}
