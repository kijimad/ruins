package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityCreation(t *testing.T) {
	t.Parallel()

	// 休息アクティビティの作成テスト
	behavior := &RestBehavior{}
	comp, err := NewActivity(behavior, 10)
	require.NoError(t, err)

	assert.Equal(t, gc.BehaviorRest, behavior.Name(), "Expected behavior to be Rest")
	assert.Equal(t, gc.ActivityStateRunning, comp.State, "Expected initial state to be Running")
	assert.Equal(t, 10, comp.Progress.Max, "Expected required 10")
	assert.Equal(t, 0, comp.Progress.Current, "Expected accumulated 0")
}

func TestActivityInfo(t *testing.T) {
	t.Parallel()
	// 休息アクティビティの情報テスト
	actorImpl := &RestBehavior{}
	info := actorImpl.Info()

	assert.Equal(t, "休息", info.Name, "Expected name '休息'")
	assert.True(t, info.Interruptible, "Expected rest activity to be interruptible")
	assert.True(t, info.Resumable, "Expected rest activity to be resumable")
}

func TestActivityInterruptAndResume(t *testing.T) {
	t.Parallel()

	comp, err := NewActivity(&RestBehavior{}, 10)
	require.NoError(t, err)

	// 初期状態での中断可能性チェック
	assert.True(t, CanInterrupt(comp), "Expected activity to be interruptible initially")

	// 中断実行
	err = Interrupt(comp, "テスト中断")
	require.NoError(t, err, "Unexpected error during interrupt")
	assert.Equal(t, gc.ActivityStatePaused, comp.State, "Expected state to be Paused after interrupt")
	assert.Equal(t, "テスト中断", comp.CancelReason, "Expected cancel reason 'テスト中断'")

	// 中断状態での再中断テスト（エラーになるはず）
	err = Interrupt(comp, "再中断")
	require.Error(t, err, "Expected error when interrupting already paused activity")

	// 再開可能性チェック
	assert.True(t, CanResume(comp), "Expected activity to be resumable")

	// 再開実行
	err = Resume(comp)
	require.NoError(t, err, "Unexpected error during resume")
	assert.Equal(t, gc.ActivityStateRunning, comp.State, "Expected state to be Running after resume")
	assert.Empty(t, comp.CancelReason, "Expected empty cancel reason after resume")
}

func TestActivityCancel(t *testing.T) {
	t.Parallel()

	comp, err := NewActivity(&WaitBehavior{}, 5)
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

	comp, err := NewActivity(&WaitBehavior{}, 5)
	require.NoError(t, err)

	// 完了実行
	Complete(comp)

	assert.Equal(t, gc.ActivityStateCompleted, comp.State, "Expected state to be Completed after complete")
	assert.True(t, IsCompleted(comp), "Expected IsCompleted() to return true")
}

func TestActivityProgressCalculation(t *testing.T) {
	t.Parallel()

	comp, err := NewActivity(&RestBehavior{}, 10)
	require.NoError(t, err)

	// 初期進捗（0%）
	progress := GetProgressPercent(comp)
	assert.Equal(t, 0.0, progress, "Expected initial progress 0%%")

	// 半分注いだ（50%）
	comp.Progress.Current = 5
	progress = GetProgressPercent(comp)
	assert.Equal(t, 50.0, progress, "Expected progress 50%%")

	// 必要量に達した（100%）
	comp.Progress.Current = 10
	progress = GetProgressPercent(comp)
	assert.Equal(t, 100.0, progress, "Expected progress 100%%")
}

func TestActivityDoTurn(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	// 長い待機の敵接近チェックは位置を前提とするため、実際のアクターと同様に座標を与える
	world.Components.GridElement.Add(actor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	behavior := &WaitBehavior{}
	comp, err := NewActivity(behavior, 3)
	require.NoError(t, err)

	// 待機は毎ターン 1 ずつ注ぐ純タイマー。Required 3 なので3ターンで完了する
	// 1ターン目
	err = behavior.DoTurn(comp, actor, world)
	require.NoError(t, err, "Unexpected error in turn 1")
	assert.Equal(t, 1, comp.Progress.Current, "Expected accumulated 1 after turn 1")
	assert.False(t, IsCompleted(comp), "Expected activity not to be completed after turn 1")

	// 2ターン目
	err = behavior.DoTurn(comp, actor, world)
	require.NoError(t, err, "Unexpected error in turn 2")
	assert.Equal(t, 2, comp.Progress.Current, "Expected accumulated 2 after turn 2")

	// 3ターン目（完了）
	err = behavior.DoTurn(comp, actor, world)
	require.NoError(t, err, "Unexpected error in turn 3")
	assert.Equal(t, 3, comp.Progress.Current, "Expected accumulated 3 after turn 3")
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

func TestNewActivityInvalidRequired(t *testing.T) {
	t.Parallel()

	t.Run("required 0は即時アクションとして正常", func(t *testing.T) {
		t.Parallel()
		comp, err := NewActivity(&WaitBehavior{}, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, comp.Progress.Max)
	})

	t.Run("負のrequiredでエラー", func(t *testing.T) {
		t.Parallel()
		_, err := NewActivity(&WaitBehavior{}, -1)
		assert.ErrorIs(t, err, ErrInvalidRequired)
	})
}

func TestGetProgressPercentEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("Required 0の場合は100%を返す", func(t *testing.T) {
		t.Parallel()
		comp := &gc.Activity{Progress: gc.IntPool{Max: 0}}
		assert.Equal(t, 100.0, GetProgressPercent(comp))
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

func TestIsCompleted_StateBased(t *testing.T) {
	t.Parallel()

	// 完了は State のみで判定する。残りターンという概念は廃止した
	running := &gc.Activity{State: gc.ActivityStateRunning}
	assert.False(t, IsCompleted(running), "Running は未完了")

	completed := &gc.Activity{State: gc.ActivityStateCompleted}
	assert.True(t, IsCompleted(completed), "Completed は完了")
}
