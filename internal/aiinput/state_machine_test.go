package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestShouldChase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ai   *gc.AI
		want bool
	}{
		{"CombatAttackは追跡する", &gc.AI{CombatCurrent: gc.CombatAttack}, true},
		{"CombatIgnoreは追跡しない", &gc.AI{CombatCurrent: gc.CombatIgnore}, false},
		{"CombatEvadeは追跡しない", &gc.AI{CombatCurrent: gc.CombatEvade}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shouldChase(tt.ai))
		})
	}
}

func TestShouldFlee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ai   *gc.AI
		want bool
	}{
		{"CombatEvadeは逃亡する", &gc.AI{CombatCurrent: gc.CombatEvade}, true},
		{"CombatAttackは逃亡しない", &gc.AI{CombatCurrent: gc.CombatAttack}, false},
		{"CombatIgnoreは逃亡しない", &gc.AI{CombatCurrent: gc.CombatIgnore}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shouldFlee(tt.ai))
		})
	}
}

func TestUpdateState_UnknownState(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		SubState:              gc.AIStateSubState("INVALID"),
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, false, 10)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState, "不明な状態は待機に初期化される")
}

func TestUpdateState_ChasingLostPlayer(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()

	ai := &gc.AI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	// プレイヤーを見失って3ターン未満は追跡継続
	rp.updateState(ai, false, 3)
	assert.Equal(t, gc.AIStateChasing, ai.SubState, "3ターン未満は追跡継続")

	// 3ターン以上見失うと移動状態へ
	rp.updateState(ai, false, 5)
	assert.Equal(t, gc.AIStateDriving, ai.SubState, "3ターン以上見失うと移動状態へ")
}

func TestUpdateState_ChasingPlayerVisible_ContinuesChase(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()

	ai := &gc.AI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	rp.updateState(ai, true, 5)
	assert.Equal(t, gc.AIStateChasing, ai.SubState, "追跡継続")
	assert.Equal(t, 1, ai.StartSubStateTurn, "StartSubStateTurnは変化しない")
}

func TestUpdateState_WaitingToDriving(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 3,
	}

	rp.updateState(ai, false, 2)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState)

	rp.updateState(ai, false, 3)
	assert.Equal(t, gc.AIStateDriving, ai.SubState)
}

func TestUpdateState_DrivingToWaiting(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, false, 4)
	assert.Equal(t, gc.AIStateDriving, ai.SubState)

	rp.updateState(ai, false, 5)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState)
}

func TestUpdateState_WaitingToChasing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, true, 1)
	assert.Equal(t, gc.AIStateChasing, ai.SubState)
}

func TestUpdateState_WaitingToFleeing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, true, 1)
	assert.Equal(t, gc.AIStateFleeing, ai.SubState)
}

func TestUpdateState_WaitingNeutralIgnoresPlayer(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		CombatDefault:         gc.CombatIgnore,
		CombatCurrent:         gc.CombatIgnore,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, true, 1)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState)
}

func TestUpdateState_DrivingToChasing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, true, 1)
	assert.Equal(t, gc.AIStateChasing, ai.SubState)
}

func TestUpdateState_DrivingToFleeing(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, true, 1)
	assert.Equal(t, gc.AIStateFleeing, ai.SubState)
}

func TestUpdateState_FleeingPlayerVisible_ResetsTurn(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, true, 3)
	assert.Equal(t, gc.AIStateFleeing, ai.SubState)
	assert.Equal(t, 3, ai.StartSubStateTurn)
}

func TestUpdateState_FleeingToDriving(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	rp.updateState(ai, false, 5)
	assert.Equal(t, gc.AIStateDriving, ai.SubState)
	assert.Equal(t, gc.CombatEvade, ai.CombatCurrent)
}

// TestCombatIgnorePermanentHostile はCombatIgnore NPCが被ダメージで永続的に敵対化することを検証する。
// 一度殴られたNPCは見失っても再発見時に追跡を再開する
func TestCombatIgnorePermanentHostile(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		CombatDefault:         gc.CombatIgnore,
		CombatCurrent:         gc.CombatIgnore,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	// 1. 初期状態: CombatIgnoreなのでプレイヤーを見ても無視する
	rp.updateState(ai, true, 1)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState, "CombatIgnoreはプレイヤーを無視する")

	// 2. 被ダメージ: ReactToHostileでCombatAttackに永続的に変化する
	ai.ReactToHostile()
	assert.Equal(t, gc.CombatAttack, ai.CombatCurrent)

	// 3. プレイヤーを発見して追跡を開始する
	rp.updateState(ai, true, 2)
	assert.Equal(t, gc.AIStateChasing, ai.SubState, "CombatAttackに変化したので追跡する")

	// 4. プレイヤーを見失い、3ターン以上経過して追跡を終了する
	rp.updateState(ai, false, 6)
	assert.Equal(t, gc.AIStateDriving, ai.SubState, "見失ってDrivingに遷移する")
	assert.Equal(t, gc.CombatAttack, ai.CombatCurrent, "敵対状態は維持される")

	// 5. Driving終了後にWaitingへ
	turnAfterDriving := ai.StartSubStateTurn + ai.DurationSubStateTurns
	rp.updateState(ai, false, turnAfterDriving)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState, "Driving終了でWaitingに戻る")

	// 6. 再発見時にまた追跡する
	rp.updateState(ai, true, turnAfterDriving+1)
	assert.Equal(t, gc.AIStateChasing, ai.SubState, "敵対が永続しているので再追跡する")
}

func TestUpdateState_ChasingTimeout(t *testing.T) {
	t.Parallel()

	rp := newRoamingPlanner()
	ai := &gc.AI{
		SubState:              gc.AIStateChasing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 10,
	}

	rp.updateState(ai, true, 10)
	assert.Equal(t, gc.AIStateWaiting, ai.SubState)
}
