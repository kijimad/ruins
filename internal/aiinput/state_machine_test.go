package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestShouldChase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		disposition *gc.Disposition
		want        bool
	}{
		{"Hostileは追跡する", &gc.Disposition{Current: gc.DispositionHostile}, true},
		{"Neutralは追跡しない", &gc.Disposition{Current: gc.DispositionNeutral}, false},
		{"Cowardlyは追跡しない", &gc.Disposition{Current: gc.DispositionCowardly}, false},
		{"Fleeingは追跡しない", &gc.Disposition{Current: gc.DispositionFleeing}, false},
		{"nilは追跡する", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shouldChase(tt.disposition))
		})
	}
}

func TestShouldFlee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		disposition *gc.Disposition
		want        bool
	}{
		{"Fleeingは逃亡する", &gc.Disposition{Current: gc.DispositionFleeing}, true},
		{"Cowardlyは逃亡する", &gc.Disposition{Current: gc.DispositionCowardly}, true},
		{"Hostileは逃亡しない", &gc.Disposition{Current: gc.DispositionHostile}, false},
		{"Neutralは逃亡しない", &gc.Disposition{Current: gc.DispositionNeutral}, false},
		{"nilは逃亡しない", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, shouldFlee(tt.disposition))
		})
	}
}

func TestUpdateState_UnknownState(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingSubState("INVALID"),
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(roaming, nil, false, 10)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState, "不明な状態は待機に初期化される")
}

func TestUpdateState_ChasingLostPlayer(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionHostile, Current: gc.DispositionHostile}

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	// プレイヤーを見失って3ターン未満は追跡継続
	sm.UpdateState(roaming, disposition, false, 3)
	assert.Equal(t, gc.AIRoamingChasing, roaming.SubState, "3ターン未満は追跡継続")

	// 3ターン以上見失うと移動状態へ
	sm.UpdateState(roaming, disposition, false, 5)
	assert.Equal(t, gc.AIRoamingDriving, roaming.SubState, "3ターン以上見失うと移動状態へ")
}

func TestUpdateState_ChasingPlayerVisible_ResetsTurn(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionHostile, Current: gc.DispositionHostile}

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 10,
	}

	// プレイヤーが見えている間はターンリセット
	sm.UpdateState(roaming, disposition, true, 5)
	assert.Equal(t, gc.AIRoamingChasing, roaming.SubState)
	assert.Equal(t, 5, roaming.StartSubStateTurn, "プレイヤー視認中はターンリセット")
}

func TestUpdateState_WaitingToDriving(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 3,
	}

	// 待機時間未満は待機のまま
	sm.UpdateState(roaming, nil, false, 2)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState)

	// 待機時間経過で移動へ
	sm.UpdateState(roaming, nil, false, 3)
	assert.Equal(t, gc.AIRoamingDriving, roaming.SubState)
}

func TestUpdateState_DrivingToWaiting(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	// 移動時間未満は移動のまま
	sm.UpdateState(roaming, nil, false, 4)
	assert.Equal(t, gc.AIRoamingDriving, roaming.SubState)

	// 移動時間経過で待機へ
	sm.UpdateState(roaming, nil, false, 5)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState)
}

func TestUpdateState_WaitingToChasing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionHostile, Current: gc.DispositionHostile}
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(roaming, disposition, true, 1)
	assert.Equal(t, gc.AIRoamingChasing, roaming.SubState)
}

func TestUpdateState_WaitingToFleeing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionCowardly, Current: gc.DispositionFleeing}
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(roaming, disposition, true, 1)
	assert.Equal(t, gc.AIRoamingFleeing, roaming.SubState)
}

func TestUpdateState_WaitingNeutralIgnoresPlayer(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionNeutral, Current: gc.DispositionNeutral}
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(roaming, disposition, true, 1)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState)
}

func TestUpdateState_DrivingToChasing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionHostile, Current: gc.DispositionHostile}
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(roaming, disposition, true, 1)
	assert.Equal(t, gc.AIRoamingChasing, roaming.SubState)
}

func TestUpdateState_DrivingToFleeing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionCowardly, Current: gc.DispositionCowardly}
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(roaming, disposition, true, 1)
	assert.Equal(t, gc.AIRoamingFleeing, roaming.SubState)
}

func TestUpdateState_FleeingPlayerVisible_ResetsTurn(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionCowardly, Current: gc.DispositionFleeing}
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(roaming, disposition, true, 3)
	assert.Equal(t, gc.AIRoamingFleeing, roaming.SubState)
	assert.Equal(t, 3, roaming.StartSubStateTurn)
}

func TestUpdateState_FleeingToDriving(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionCowardly, Current: gc.DispositionFleeing}
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingFleeing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 5,
	}

	sm.UpdateState(roaming, disposition, false, 5)
	assert.Equal(t, gc.AIRoamingDriving, roaming.SubState)
	assert.Equal(t, gc.DispositionCowardly, disposition.Current)
}

func TestUpdateState_ChasingTimeout(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingChasing,
		StartSubStateTurn:     0,
		DurationSubStateTurns: 10,
	}

	sm.UpdateState(roaming, nil, true, 10)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState)
}
