package actions

import (
	"testing"

	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/testutil"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestActivityManagerCreation(t *testing.T) {
	t.Parallel()
	manager := NewActivityManager(logger.New(logger.CategoryAction))

	if manager == nil {
		t.Errorf("Expected non-nil activity manager")
		return
	}

	if manager.currentActivities == nil {
		t.Errorf("Expected non-nil current activities map")
		return
	}

	if len(manager.currentActivities) != 0 {
		t.Errorf("Expected empty activities map initially, got %d activities", len(manager.currentActivities))
	}
}

func TestActivityManagerStartActivity(t *testing.T) {
	t.Parallel()
	manager := NewActivityManager(logger.New(logger.CategoryAction))
	world := testutil.InitTestWorld(t)
	actor := ecs.Entity(1)

	// アクティビティを作成
	actorImpl := &WaitActivity{}
	activity := NewActivity(actorImpl, actor, 5)

	// アクティビティ開始
	err := manager.StartActivity(activity, world)
	if err != nil {
		t.Errorf("Unexpected error starting activity: %v", err)
	}

	// アクティビティが登録されているかチェック
	currentActivity := manager.GetCurrentActivity(actor)
	if currentActivity == nil {
		t.Errorf("Expected activity to be registered")
	}

	if currentActivity != activity {
		t.Errorf("Expected registered activity to match started activity")
	}

	// HasActivity のテスト
	if !manager.HasActivity(actor) {
		t.Errorf("Expected HasActivity to return true")
	}

	// 存在しないエンティティのテスト
	nonExistentActor := ecs.Entity(999)
	if manager.HasActivity(nonExistentActor) {
		t.Errorf("Expected HasActivity to return false for non-existent entity")
	}
}

func TestActivityManagerMultipleActivities(t *testing.T) {
	t.Parallel()
	manager := NewActivityManager(logger.New(logger.CategoryAction))
	world := testutil.InitTestWorld(t)

	actor1 := ecs.Entity(1)
	actor2 := ecs.Entity(2)

	// 複数のアクターでアクティビティを開始
	actorImpl := &WaitActivity{}
	activity1 := NewActivity(actorImpl, actor1, 10)
	activity2 := NewActivity(actorImpl, actor2, 5)

	err := manager.StartActivity(activity1, world)
	if err != nil {
		t.Errorf("Unexpected error starting activity 1: %v", err)
	}

	err = manager.StartActivity(activity2, world)
	if err != nil {
		t.Errorf("Unexpected error starting activity 2: %v", err)
	}

	// 両方のアクティビティが登録されているかチェック
	if !manager.HasActivity(actor1) {
		t.Errorf("Expected actor1 to have activity")
	}

	if !manager.HasActivity(actor2) {
		t.Errorf("Expected actor2 to have activity")
	}

	// 正しいアクティビティが取得できるかチェック
	retrievedActivity1 := manager.GetCurrentActivity(actor1)
	if retrievedActivity1 == nil {
		t.Errorf("Expected actor1 to have activity")
	}

	retrievedActivity2 := manager.GetCurrentActivity(actor2)
	if retrievedActivity2 == nil {
		t.Errorf("Expected actor2 to have activity")
	}
}

func TestActivityManagerReplaceActivity(t *testing.T) {
	t.Parallel()
	manager := NewActivityManager(logger.New(logger.CategoryAction))
	world := testutil.InitTestWorld(t)
	actor := ecs.Entity(1)

	// 最初のアクティビティを開始
	actorImpl := &WaitActivity{}
	activity1 := NewActivity(actorImpl, actor, 10)
	err := manager.StartActivity(activity1, world)
	if err != nil {
		t.Errorf("Unexpected error starting first activity: %v", err)
	}

	// 最初のアクティビティが実行中であることを確認
	if activity1.State != ActivityStateRunning {
		t.Errorf("Expected first activity to be running")
	}

	// 新しいアクティビティを開始（古いものを置き換え）
	activity2 := NewActivity(actorImpl, actor, 5)
	err = manager.StartActivity(activity2, world)
	if err != nil {
		t.Errorf("Unexpected error starting second activity: %v", err)
	}

	// 古いアクティビティが中断されているかチェック
	if activity1.State != ActivityStatePaused {
		t.Errorf("Expected first activity to be paused after replacement, got %v", activity1.State)
	}

	// 新しいアクティビティが現在のアクティビティになっているかチェック
	currentActivity := manager.GetCurrentActivity(actor)
	if currentActivity != activity2 {
		t.Errorf("Expected current activity to be the second activity")
	}
}

func TestActivityManagerInterruptAndResume(t *testing.T) {
	t.Parallel()
	manager := NewActivityManager(logger.New(logger.CategoryAction))
	world := testutil.InitTestWorld(t)
	actor := ecs.Entity(1)

	// アクティビティを開始
	actorImpl := &WaitActivity{}
	activity := NewActivity(actorImpl, actor, 10)
	err := manager.StartActivity(activity, world)
	if err != nil {
		t.Errorf("Unexpected error starting activity: %v", err)
	}

	// アクティビティを中断
	err = manager.InterruptActivity(actor, "テスト中断")
	if err != nil {
		t.Errorf("Unexpected error interrupting activity: %v", err)
	}

	if activity.State != ActivityStatePaused {
		t.Errorf("Expected activity to be paused after interrupt")
	}

	// 中断されたアクティビティはアクティブではない
	if manager.HasActivity(actor) {
		t.Errorf("Expected HasActivity to return false for paused activity")
	}

	// アクティビティを再開
	err = manager.ResumeActivity(actor, world)
	if err != nil {
		t.Errorf("Unexpected error resuming activity: %v", err)
	}

	if activity.State != ActivityStateRunning {
		t.Errorf("Expected activity to be running after resume")
	}

	// 再開されたアクティビティはアクティブ
	if !manager.HasActivity(actor) {
		t.Errorf("Expected HasActivity to return true for resumed activity")
	}

	// 存在しないアクティビティの中断・再開テスト
	nonExistentActor := ecs.Entity(999)
	err = manager.InterruptActivity(nonExistentActor, "テスト")
	if err == nil {
		t.Errorf("Expected error when interrupting non-existent activity")
	}

	err = manager.ResumeActivity(nonExistentActor, world)
	if err == nil {
		t.Errorf("Expected error when resuming non-existent activity")
	}
}

func TestActivityManagerCancel(t *testing.T) {
	t.Parallel()
	manager := NewActivityManager(logger.New(logger.CategoryAction))
	world := testutil.InitTestWorld(t)
	actor := ecs.Entity(1)

	// アクティビティを開始
	actorImpl := &WaitActivity{}
	activity := NewActivity(actorImpl, actor, 5)
	err := manager.StartActivity(activity, world)
	if err != nil {
		t.Errorf("Unexpected error starting activity: %v", err)
	}

	// アクティビティをキャンセル
	manager.CancelActivity(actor, "テストキャンセル", world)

	if activity.State != ActivityStateCanceled {
		t.Errorf("Expected activity to be canceled")
	}

	// キャンセルされたアクティビティは管理対象から削除される
	currentActivity := manager.GetCurrentActivity(actor)
	if currentActivity != nil {
		t.Errorf("Expected no current activity after cancel")
	}

	// 存在しないアクティビティのキャンセル（エラーにならない）
	nonExistentActor := ecs.Entity(999)
	manager.CancelActivity(nonExistentActor, "テスト", world) // パニックしないことを確認
}

func TestActivityManagerProcessTurn(t *testing.T) {
	t.Parallel()
	manager := NewActivityManager(logger.New(logger.CategoryAction))
	world := testutil.InitTestWorld(t)

	actor1 := ecs.Entity(1)
	actor2 := ecs.Entity(2)

	// 短いアクティビティと長いアクティビティを開始
	actorImpl := &WaitActivity{}
	shortActivity := NewActivity(actorImpl, actor1, 2) // 2ターンで完了
	longActivity := NewActivity(actorImpl, actor2, 5)  // 5ターンで完了

	err := manager.StartActivity(shortActivity, world)
	if err != nil {
		t.Errorf("Unexpected error starting short activity: %v", err)
	}
	err = manager.StartActivity(longActivity, world)
	if err != nil {
		t.Errorf("Unexpected error starting long activity: %v", err)
	}

	// 初期状態の確認
	summary := manager.GetActivitySummary()
	if summary["total"] != 2 {
		t.Errorf("Expected 2 total activities, got %v", summary["total"])
	}
	if summary["active"] != 2 {
		t.Errorf("Expected 2 active activities, got %v", summary["active"])
	}

	// 1ターン目処理
	manager.ProcessTurn(world)

	// 両方まだ実行中
	if shortActivity.TurnsLeft != 1 {
		t.Errorf("Expected short activity to have 1 turn left, got %d", shortActivity.TurnsLeft)
	}
	if longActivity.TurnsLeft != 4 {
		t.Errorf("Expected long activity to have 4 turns left, got %d", longActivity.TurnsLeft)
	}

	// 2ターン目処理
	manager.ProcessTurn(world)

	// 短いアクティビティが完了
	if !shortActivity.IsCompleted() {
		t.Errorf("Expected short activity to be completed")
	}
	if longActivity.TurnsLeft != 3 {
		t.Errorf("Expected long activity to have 3 turns left, got %d", longActivity.TurnsLeft)
	}

	// 完了したアクティビティは管理対象から削除される
	if manager.GetCurrentActivity(actor1) != nil {
		t.Errorf("Expected completed activity to be removed")
	}
	if manager.GetCurrentActivity(actor2) == nil {
		t.Errorf("Expected long activity to still be present")
	}

	// サマリーの確認
	summary = manager.GetActivitySummary()
	if summary["total"] != 1 {
		t.Errorf("Expected 1 total activity after completion, got %v", summary["total"])
	}
}

func TestActivityManagerSummary(t *testing.T) {
	t.Parallel()
	manager := NewActivityManager(logger.New(logger.CategoryAction))
	world := testutil.InitTestWorld(t)

	// 初期状態のサマリー
	summary := manager.GetActivitySummary()
	if summary["total"] != 0 {
		t.Errorf("Expected 0 total activities initially, got %v", summary["total"])
	}
	if summary["active"] != 0 {
		t.Errorf("Expected 0 active activities initially, got %v", summary["active"])
	}
	if summary["paused"] != 0 {
		t.Errorf("Expected 0 paused activities initially, got %v", summary["paused"])
	}

	// アクティビティを追加
	actor1 := ecs.Entity(1)
	actor2 := ecs.Entity(2)

	actorImpl := &WaitActivity{}
	activity1 := NewActivity(actorImpl, actor1, 10)
	activity2 := NewActivity(actorImpl, actor2, 5)

	err := manager.StartActivity(activity1, world)
	if err != nil {
		t.Errorf("Unexpected error starting activity1: %v", err)
	}
	err = manager.StartActivity(activity2, world)
	if err != nil {
		t.Errorf("Unexpected error starting activity2: %v", err)
	}

	// 1つを中断
	err = manager.InterruptActivity(actor1, "テスト")
	if err != nil {
		t.Errorf("Unexpected error interrupting activity: %v", err)
	}

	// サマリーの確認
	summary = manager.GetActivitySummary()
	if summary["total"] != 2 {
		t.Errorf("Expected 2 total activities, got %v", summary["total"])
	}
	if summary["active"] != 1 {
		t.Errorf("Expected 1 active activity, got %v", summary["active"])
	}
	if summary["paused"] != 1 {
		t.Errorf("Expected 1 paused activity, got %v", summary["paused"])
	}
}
