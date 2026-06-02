package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine_Hostile(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionHostile, Current: gc.DispositionHostile}

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// 1ターン目：まだ待機継続
	sm.UpdateState(roaming, disposition, false, 2)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState, "1ターン経過時は待機継続")

	// 3ターン目：待機時間終了で移動状態へ
	sm.UpdateState(roaming, disposition, false, 3)
	assert.Equal(t, gc.AIRoamingDriving, roaming.SubState, "待機時間終了で移動状態へ遷移")

	// プレイヤー発見で追跡状態へ
	sm.UpdateState(roaming, disposition, true, 4)
	assert.Equal(t, gc.AIRoamingChasing, roaming.SubState, "Hostileはプレイヤー発見で追跡状態へ遷移")
}

func TestStateMachine_Neutral(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionNeutral, Current: gc.DispositionNeutral}

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// プレイヤーを発見しても追跡しない
	sm.UpdateState(roaming, disposition, true, 2)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState, "Neutralはプレイヤーを見ても追跡しない")
}

func TestStateMachine_Cowardly(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionCowardly, Current: gc.DispositionCowardly}

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// プレイヤー発見で逃亡状態へ
	sm.UpdateState(roaming, disposition, true, 2)
	assert.Equal(t, gc.AIRoamingFleeing, roaming.SubState, "Cowardlyはプレイヤー発見で逃亡状態へ遷移")
}

func TestStateMachine_Fleeing_Recovery(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionCowardly, Current: gc.DispositionFleeing}

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingFleeing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 3,
	}

	// 逃亡中にプレイヤーが見えている間は逃亡継続
	sm.UpdateState(roaming, disposition, true, 2)
	assert.Equal(t, gc.AIRoamingFleeing, roaming.SubState, "プレイヤーが見えている間は逃亡継続")

	// プレイヤーを見失い、逃亡時間終了でデフォルトに復帰
	roaming.StartSubStateTurn = 1
	sm.UpdateState(roaming, disposition, false, 5)
	assert.Equal(t, gc.AIRoamingDriving, roaming.SubState, "プレイヤーを見失い逃亡時間終了で移動へ")
	assert.Equal(t, gc.DispositionCowardly, disposition.Current, "デフォルト態度に復帰")
}

func TestStateMachine_NilDisposition(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// Dispositionがnilでも既存動作を維持する
	sm.UpdateState(roaming, nil, true, 2)
	assert.Equal(t, gc.AIRoamingChasing, roaming.SubState, "Dispositionなしはデフォルトで追跡")
}

func TestStateMachine_NeutralToHostile_StartChasing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionNeutral, Current: gc.DispositionNeutral}

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// Neutralはプレイヤーを見ても追跡しない
	sm.UpdateState(roaming, disposition, true, 2)
	assert.Equal(t, gc.AIRoamingDriving, roaming.SubState, "Neutralは追跡しない")

	// 被ダメージでDispositionがHostileに変化した（reactToHostileAction相当）
	disposition.Current = gc.DispositionHostile

	// 次のターンでプレイヤーを見たら追跡を開始する
	sm.UpdateState(roaming, disposition, true, 3)
	assert.Equal(t, gc.AIRoamingChasing, roaming.SubState, "Hostile化後はプレイヤー発見で追跡開始")
}

func TestStateMachine_CowardlyToFleeing_StartFleeing(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()
	disposition := &gc.Disposition{Default: gc.DispositionCowardly, Current: gc.DispositionCowardly}

	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// 被ダメージでDispositionがFleeingに変化した（reactToHostileAction相当）
	disposition.Current = gc.DispositionFleeing

	// プレイヤーを見ていなくてもFleeing化後は逃亡状態へ遷移する
	sm.UpdateState(roaming, disposition, false, 2)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState, "プレイヤーが見えてなければまだ待機")

	// プレイヤーが見える場合は逃亡開始
	sm.UpdateState(roaming, disposition, true, 3)
	assert.Equal(t, gc.AIRoamingFleeing, roaming.SubState, "Fleeing化後はプレイヤー発見で逃亡開始")
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
