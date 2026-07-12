package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestUpdateState_UnknownState(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		SubState:              gc.AIStateSubState("INVALID"),
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, false, 10)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState, "不明な状態は待機に初期化される")
}

func TestUpdateState_ChasingLostPlayer(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())

	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	// プレイヤーを見失って3ターン未満は追跡継続
	rp.updateState(solo, false, 3)
	assert.Equal(t, gc.AIStateChasing, solo.SubState, "3ターン未満は追跡継続")

	// 3ターン以上見失うと移動状態へ
	rp.updateState(solo, false, 5)
	assert.Equal(t, gc.AIStateDriving, solo.SubState, "3ターン以上見失うと移動状態へ")
}

func TestUpdateState_ChasingPlayerVisible_ContinuesChase(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())

	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	rp.updateState(solo, true, 5)
	assert.Equal(t, gc.AIStateChasing, solo.SubState, "追跡継続")
	assert.Equal(t, 1, solo.StartSubStateTurn, "StartSubStateTurnは変化しない")
}

func TestUpdateState_WaitingToDriving(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 3,
	}

	rp.updateState(solo, false, 2)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState)

	rp.updateState(solo, false, 3)
	assert.Equal(t, gc.AIStateDriving, solo.SubState)
}

func TestUpdateState_DrivingToWaiting(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, false, 4)
	assert.Equal(t, gc.AIStateDriving, solo.SubState)

	rp.updateState(solo, false, 5)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState)
}

func TestUpdateState_WaitingToChasing(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, true, 1)
	assert.Equal(t, gc.AIStateChasing, solo.SubState)
}

func TestUpdateState_WaitingToFleeing(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, true, 1)
	assert.Equal(t, gc.AIStateFleeing, solo.SubState)
}

func TestUpdateState_WaitingNeutralIgnoresPlayer(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatIgnore,
		CombatCurrent:         gc.CombatIgnore,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, true, 1)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState)
}

func TestUpdateState_DrivingToChasing(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, true, 1)
	assert.Equal(t, gc.AIStateChasing, solo.SubState)
}

func TestUpdateState_DrivingToFleeing(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, true, 1)
	assert.Equal(t, gc.AIStateFleeing, solo.SubState)
}

func TestUpdateState_FleeingPlayerVisible_ResetsTurn(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, true, 3)
	assert.Equal(t, gc.AIStateFleeing, solo.SubState)
	assert.Equal(t, 3, solo.StartSubStateTurn)
}

func TestUpdateState_FleeingToDriving(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, false, 5)
	assert.Equal(t, gc.AIStateDriving, solo.SubState)
	assert.Equal(t, gc.CombatEvade, solo.CombatCurrent)
}

// TestCombatIgnorePermanentHostile はCombatIgnore NPCが被ダメージで永続的に敵対化することを検証する。
// 一度殴られたNPCは見失っても再発見時に追跡を再開する
func TestCombatIgnorePermanentHostile(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatIgnore,
		CombatCurrent:         gc.CombatIgnore,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	// 1. 初期状態: CombatIgnoreなのでプレイヤーを見ても無視する
	rp.updateState(solo, true, 1)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState, "CombatIgnoreはプレイヤーを無視する")

	// 2. 被ダメージ: ReactToHostileでCombatAttackに永続的に変化する
	solo.ReactToHostile()
	assert.Equal(t, gc.CombatAttack, solo.CombatCurrent)

	// 3. プレイヤーを発見して追跡を開始する
	rp.updateState(solo, true, 2)
	assert.Equal(t, gc.AIStateChasing, solo.SubState, "CombatAttackに変化したので追跡する")

	// 4. プレイヤーを見失い、3ターン以上経過して追跡を終了する
	rp.updateState(solo, false, 6)
	assert.Equal(t, gc.AIStateDriving, solo.SubState, "見失ってDrivingに遷移する")
	assert.Equal(t, gc.CombatAttack, solo.CombatCurrent, "敵対状態は維持される")

	// 5. Driving終了後にWaitingへ
	turnAfterDriving := solo.StartSubStateTurn + solo.DurationSubStateTurns
	rp.updateState(solo, false, turnAfterDriving)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState, "Driving終了でWaitingに戻る")

	// 6. 再発見時にまた追跡する
	rp.updateState(solo, true, turnAfterDriving+1)
	assert.Equal(t, gc.AIStateChasing, solo.SubState, "敵対が永続しているので再追跡する")
}

func TestUpdateState_ChasingTimeout(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())
	solo := &gc.SoloAI{
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 10,
	}

	rp.updateState(solo, true, 10)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState)
}
