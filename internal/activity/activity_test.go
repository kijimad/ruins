package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestActivityCreation(t *testing.T) {
	t.Parallel()

	// 休息アクティビティの作成テスト
	behavior := &RestActivity{}
	comp, err := NewActivity(behavior, 10)
	require.NoError(t, err)

	assert.Equal(t, gc.BehaviorRest, behavior.Name(), "Expected behavior to be Rest")
	assert.Equal(t, gc.ActivityStateRunning, comp.State, "Expected initial state to be Running")
	assert.Equal(t, 10, comp.TurnsTotal, "Expected turns total 10")
	assert.Equal(t, 10, comp.TurnsLeft, "Expected turns left 10")
}

func TestActivityInfo(t *testing.T) {
	t.Parallel()
	// 休息アクティビティの情報テスト
	actorImpl := &RestActivity{}
	info := actorImpl.Info()

	assert.Equal(t, "休息", info.Name, "Expected name '休息'")
	assert.True(t, info.Interruptible, "Expected rest activity to be interruptible")
	assert.True(t, info.Resumable, "Expected rest activity to be resumable")
}

func TestActivityInterruptAndResume(t *testing.T) {
	t.Parallel()

	comp, err := NewActivity(&RestActivity{}, 10)
	require.NoError(t, err)

	// 初期状態での中断可能性チェック
	assert.True(t, CanInterrupt(comp), "Expected activity to be interruptible initially")

	// 中断実行
	err = Interrupt(comp, "テスト中断")
	assert.NoError(t, err, "Unexpected error during interrupt")
	assert.Equal(t, gc.ActivityStatePaused, comp.State, "Expected state to be Paused after interrupt")
	assert.Equal(t, "テスト中断", comp.CancelReason, "Expected cancel reason 'テスト中断'")

	// 中断状態での再中断テスト（エラーになるはず）
	err = Interrupt(comp, "再中断")
	assert.Error(t, err, "Expected error when interrupting already paused activity")

	// 再開可能性チェック
	assert.True(t, CanResume(comp), "Expected activity to be resumable")

	// 再開実行
	err = Resume(comp)
	assert.NoError(t, err, "Unexpected error during resume")
	assert.Equal(t, gc.ActivityStateRunning, comp.State, "Expected state to be Running after resume")
	assert.Equal(t, "", comp.CancelReason, "Expected empty cancel reason after resume")
}

func TestActivityCancel(t *testing.T) {
	t.Parallel()

	comp, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)

	// キャンセル前はIsCanceledがfalse
	assert.False(t, IsCanceled(comp), "Expected IsCanceled to be false before cancel")

	// キャンセル実行
	Cancel(comp, "テストキャンセル")

	assert.Equal(t, gc.ActivityStateCanceled, comp.State, "Expected state to be Canceled after cancel")
	assert.Equal(t, "テストキャンセル", comp.CancelReason, "Expected cancel reason 'テストキャンセル'")
	assert.True(t, IsCanceled(comp), "Expected IsCanceled to be true after cancel")

	// キャンセル後は中断・再開不可
	assert.False(t, CanInterrupt(comp), "Expected canceled activity to not be interruptible")
	assert.False(t, CanResume(comp), "Expected canceled activity to not be resumable")
}

func TestActivityComplete(t *testing.T) {
	t.Parallel()

	comp, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)

	// 完了実行
	Complete(comp)

	assert.Equal(t, gc.ActivityStateCompleted, comp.State, "Expected state to be Completed after complete")
	assert.Equal(t, 0, comp.TurnsLeft, "Expected turns left 0 after complete")
	assert.True(t, IsCompleted(comp), "Expected IsCompleted() to return true")
}

func TestActivityProgressCalculation(t *testing.T) {
	t.Parallel()

	comp, err := NewActivity(&RestActivity{}, 10)
	require.NoError(t, err)

	// 初期進捗（0%）
	progress := GetProgressPercent(comp)
	assert.Equal(t, 0.0, progress, "Expected initial progress 0%%")

	// 5ターン進行（50%）
	comp.TurnsLeft = 5
	progress = GetProgressPercent(comp)
	assert.Equal(t, 50.0, progress, "Expected progress 50%%")

	// 完了（100%）
	comp.TurnsLeft = 0
	progress = GetProgressPercent(comp)
	assert.Equal(t, 100.0, progress, "Expected progress 100%%")
}

func TestActivityDoTurn(t *testing.T) {
	t.Parallel()

	actor := ecs.Entity(1)
	behavior := &WaitActivity{}
	comp, err := NewActivity(behavior, 3)
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)

	// 1ターン目
	err = behavior.DoTurn(comp, actor, world)
	assert.NoError(t, err, "Unexpected error in turn 1")
	assert.Equal(t, 2, comp.TurnsLeft, "Expected 2 turns left after turn 1")
	assert.False(t, IsCompleted(comp), "Expected activity not to be completed after turn 1")

	// 2ターン目
	err = behavior.DoTurn(comp, actor, world)
	assert.NoError(t, err, "Unexpected error in turn 2")
	assert.Equal(t, 1, comp.TurnsLeft, "Expected 1 turn left after turn 2")

	// 3ターン目（完了）
	err = behavior.DoTurn(comp, actor, world)
	assert.NoError(t, err, "Unexpected error in turn 3")
	assert.Equal(t, 0, comp.TurnsLeft, "Expected 0 turns left after turn 3")
	assert.True(t, IsCompleted(comp), "Expected activity to be completed after turn 3")
}

func TestGetBehavior(t *testing.T) {
	t.Parallel()

	t.Run("登録済みBehaviorを取得できる", func(t *testing.T) {
		t.Parallel()
		behavior, err := GetBehavior(gc.BehaviorWait)
		require.NoError(t, err)
		assert.Equal(t, gc.BehaviorWait, behavior.Name())
	})

	t.Run("未登録Behaviorでエラーを返す", func(t *testing.T) {
		t.Parallel()
		_, err := GetBehavior(gc.BehaviorName("Unknown"))
		assert.Error(t, err)
	})
}

func TestNewActivityInvalidDuration(t *testing.T) {
	t.Parallel()

	t.Run("duration 0でエラー", func(t *testing.T) {
		t.Parallel()
		_, err := NewActivity(&WaitActivity{}, 0)
		assert.Error(t, err)
	})

	t.Run("負のdurationでエラー", func(t *testing.T) {
		t.Parallel()
		_, err := NewActivity(&WaitActivity{}, -1)
		assert.Error(t, err)
	})
}

func TestGetProgressPercentEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("TurnsTotal 0の場合は100%を返す", func(t *testing.T) {
		t.Parallel()
		comp := &gc.Activity{TurnsTotal: 0, TurnsLeft: 0}
		assert.Equal(t, 100.0, GetProgressPercent(comp))
	})
}

func TestCalculateRequiredTurns(t *testing.T) {
	t.Parallel()

	t.Run("APベースの計算", func(t *testing.T) {
		t.Parallel()
		behavior := &RestActivity{}
		info := behavior.Info()
		if info.TotalRequiredAP > 0 {
			turns := CalculateRequiredTurns(behavior, 100)
			assert.GreaterOrEqual(t, turns, 1)
		}
	})

	t.Run("APとcharacterAPで正しく切り上げ除算する", func(t *testing.T) {
		t.Parallel()
		behavior := &WaitActivity{} // TotalRequiredAP=500
		turns := CalculateRequiredTurns(behavior, 100)
		assert.Equal(t, 5, turns) // 500 / 100 = 5
	})

	t.Run("characterAPが0の場合は1を返す", func(t *testing.T) {
		t.Parallel()
		behavior := &RestActivity{}
		turns := CalculateRequiredTurns(behavior, 0)
		assert.Equal(t, 1, turns)
	})
}

func TestGetDisplayName(t *testing.T) {
	t.Parallel()

	t.Run("登録済みBehaviorの名前を返す", func(t *testing.T) {
		t.Parallel()
		comp := &gc.Activity{BehaviorName: gc.BehaviorWait}
		name := GetDisplayName(comp)
		assert.Equal(t, "待機", name)
	})

	t.Run("未登録BehaviorはBehaviorName文字列を返す", func(t *testing.T) {
		t.Parallel()
		comp := &gc.Activity{BehaviorName: gc.BehaviorName("Unknown")}
		name := GetDisplayName(comp)
		assert.Equal(t, "Unknown", name)
	})
}

func TestIsActive(t *testing.T) {
	t.Parallel()

	t.Run("Running状態はアクティブ", func(t *testing.T) {
		t.Parallel()
		comp := &gc.Activity{State: gc.ActivityStateRunning}
		assert.True(t, IsActive(comp))
	})

	t.Run("Paused状態は非アクティブ", func(t *testing.T) {
		t.Parallel()
		comp := &gc.Activity{State: gc.ActivityStatePaused}
		assert.False(t, IsActive(comp))
	})

	t.Run("Completed状態は非アクティブ", func(t *testing.T) {
		t.Parallel()
		comp := &gc.Activity{State: gc.ActivityStateCompleted}
		assert.False(t, IsActive(comp))
	})
}

func TestIsCompleted_TurnsLeftZero(t *testing.T) {
	t.Parallel()

	comp := &gc.Activity{State: gc.ActivityStateRunning, TurnsLeft: 0}
	assert.True(t, IsCompleted(comp), "TurnsLeftが0ならCompletedとみなす")
}
