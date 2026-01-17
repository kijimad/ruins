package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine()

	// テストケース：待機状態からの遷移
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2, // 2ターンの待機時間
	}

	// 1ターン目：まだ待機継続
	sm.UpdateState(roaming, false, 2)
	assert.Equal(t, gc.AIRoamingWaiting, roaming.SubState, "1ターン経過時は待機継続")

	// 3ターン目：待機時間終了で移動状態へ
	sm.UpdateState(roaming, false, 3)
	assert.Equal(t, gc.AIRoamingDriving, roaming.SubState, "待機時間終了で移動状態へ遷移")

	// プレイヤー発見で追跡状態へ
	sm.UpdateState(roaming, true, 4)
	assert.Equal(t, gc.AIRoamingChasing, roaming.SubState, "プレイヤー発見で追跡状態へ遷移")

	t.Logf("状態遷移テスト完了")
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
