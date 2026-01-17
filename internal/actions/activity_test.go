package actions

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestActivityCreation(t *testing.T) {
	t.Parallel()
	actor := ecs.Entity(1)

	// 休息アクティビティの作成テスト
	actorImpl := &RestActivity{}
	activity := NewActivity(actorImpl, actor, 10)

	if activity.ActorImpl != actorImpl {
		t.Errorf("Expected activity actor %v, got %v", actorImpl, activity.ActorImpl)
	}

	if activity.State != ActivityStateRunning {
		t.Errorf("Expected initial state %v, got %v", ActivityStateRunning, activity.State)
	}

	if activity.TurnsTotal != 10 {
		t.Errorf("Expected turns total 10, got %d", activity.TurnsTotal)
	}

	if activity.TurnsLeft != 10 {
		t.Errorf("Expected turns left 10, got %d", activity.TurnsLeft)
	}

	if activity.Actor != actor {
		t.Errorf("Expected actor %v, got %v", actor, activity.Actor)
	}
}

func TestActivityInfo(t *testing.T) {
	t.Parallel()
	// 休息アクティビティの情報テスト
	actorImpl := &RestActivity{}
	info := actorImpl.Info()

	if info.Name != "休息" {
		t.Errorf("Expected name '休息', got '%s'", info.Name)
	}

	if info.Name != "休息" {
		t.Errorf("Expected name '休息', got '%s'", info.Name)
	}

	if !info.Interruptible {
		t.Errorf("Expected rest activity to be interruptible")
	}

	if !info.Resumable {
		t.Errorf("Expected rest activity to be resumable")
	}
}

func TestActivityInterruptAndResume(t *testing.T) {
	t.Parallel()
	actor := ecs.Entity(1)
	actorImpl := &RestActivity{}
	activity := NewActivity(actorImpl, actor, 10)

	// 初期状態での中断可能性チェック
	if !activity.CanInterrupt() {
		t.Errorf("Expected activity to be interruptible initially")
	}

	// 中断実行
	err := activity.Interrupt("テスト中断")
	if err != nil {
		t.Errorf("Unexpected error during interrupt: %v", err)
	}

	if activity.State != ActivityStatePaused {
		t.Errorf("Expected state %v after interrupt, got %v", ActivityStatePaused, activity.State)
	}

	if activity.CancelReason != "テスト中断" {
		t.Errorf("Expected cancel reason 'テスト中断', got '%s'", activity.CancelReason)
	}

	// 中断状態での再中断テスト（エラーになるはず）
	err = activity.Interrupt("再中断")
	if err == nil {
		t.Errorf("Expected error when interrupting already paused activity")
	}

	// 再開可能性チェック
	if !activity.CanResume() {
		t.Errorf("Expected activity to be resumable")
	}

	// 再開実行
	err = activity.Resume()
	if err != nil {
		t.Errorf("Unexpected error during resume: %v", err)
	}

	if activity.State != ActivityStateRunning {
		t.Errorf("Expected state %v after resume, got %v", ActivityStateRunning, activity.State)
	}

	if activity.CancelReason != "" {
		t.Errorf("Expected empty cancel reason after resume, got '%s'", activity.CancelReason)
	}
}

func TestActivityCancel(t *testing.T) {
	t.Parallel()
	actor := ecs.Entity(1)
	actorImpl := &WaitActivity{}
	activity := NewActivity(actorImpl, actor, 5)

	// キャンセル実行
	activity.Cancel("テストキャンセル")

	if activity.State != ActivityStateCanceled {
		t.Errorf("Expected state %v after cancel, got %v", ActivityStateCanceled, activity.State)
	}

	if activity.CancelReason != "テストキャンセル" {
		t.Errorf("Expected cancel reason 'テストキャンセル', got '%s'", activity.CancelReason)
	}

	// キャンセル後は中断・再開不可
	if activity.CanInterrupt() {
		t.Errorf("Expected canceled activity to not be interruptible")
	}

	if activity.CanResume() {
		t.Errorf("Expected canceled activity to not be resumable")
	}
}

func TestActivityComplete(t *testing.T) {
	t.Parallel()
	actor := ecs.Entity(1)
	actorImpl := &WaitActivity{}
	activity := NewActivity(actorImpl, actor, 5)

	// 完了実行
	activity.Complete()

	if activity.State != ActivityStateCompleted {
		t.Errorf("Expected state %v after complete, got %v", ActivityStateCompleted, activity.State)
	}

	if activity.TurnsLeft != 0 {
		t.Errorf("Expected turns left 0 after complete, got %d", activity.TurnsLeft)
	}

	if !activity.IsCompleted() {
		t.Errorf("Expected IsCompleted() to return true")
	}
}

func TestActivityProgressCalculation(t *testing.T) {
	t.Parallel()
	actor := ecs.Entity(1)
	actorImpl := &RestActivity{}
	activity := NewActivity(actorImpl, actor, 10)

	// 初期進捗（0%）
	progress := activity.GetProgressPercent()
	if progress != 0.0 {
		t.Errorf("Expected initial progress 0%%, got %f%%", progress)
	}

	// 5ターン進行（50%）
	activity.TurnsLeft = 5
	progress = activity.GetProgressPercent()
	if progress != 50.0 {
		t.Errorf("Expected progress 50%%, got %f%%", progress)
	}

	// 完了（100%）
	activity.TurnsLeft = 0
	progress = activity.GetProgressPercent()
	if progress != 100.0 {
		t.Errorf("Expected progress 100%%, got %f%%", progress)
	}
}

func TestActivityDoTurn(t *testing.T) {
	t.Parallel()

	actor := ecs.Entity(1)
	actorImpl := &WaitActivity{}
	activity := NewActivity(actorImpl, actor, 3)

	world := testutil.InitTestWorld(t)

	// 1ターン目
	err := activity.ActorImpl.DoTurn(activity, world)
	if err != nil {
		t.Errorf("Unexpected error in turn 1: %v", err)
	}

	if activity.TurnsLeft != 2 {
		t.Errorf("Expected 2 turns left after turn 1, got %d", activity.TurnsLeft)
	}

	if activity.IsCompleted() {
		t.Errorf("Expected activity not to be completed after turn 1")
	}

	// 2ターン目
	err = activity.ActorImpl.DoTurn(activity, world)
	if err != nil {
		t.Errorf("Unexpected error in turn 2: %v", err)
	}

	if activity.TurnsLeft != 1 {
		t.Errorf("Expected 1 turn left after turn 2, got %d", activity.TurnsLeft)
	}

	// 3ターン目（完了）
	err = activity.ActorImpl.DoTurn(activity, world)
	if err != nil {
		t.Errorf("Unexpected error in turn 3: %v", err)
	}

	if activity.TurnsLeft != 0 {
		t.Errorf("Expected 0 turns left after turn 3, got %d", activity.TurnsLeft)
	}

	if !activity.IsCompleted() {
		t.Errorf("Expected activity to be completed after turn 3")
	}
}
