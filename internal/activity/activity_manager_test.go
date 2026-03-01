package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// getActivitySummary はテスト用にアクティビティの要約情報を取得する
func getActivitySummary(t *testing.T, world w.World) map[string]int {
	t.Helper()
	summary := map[string]int{
		"total":  0,
		"active": 0,
		"paused": 0,
	}

	world.Manager.Join(world.Components.Activity).Visit(ecs.Visit(func(entity ecs.Entity) {
		comp := world.Components.Activity.Get(entity).(*gc.Activity)
		summary["total"]++
		switch comp.State {
		case gc.ActivityStateRunning:
			summary["active"]++
		case gc.ActivityStatePaused:
			summary["paused"]++
		case gc.ActivityStateCompleted, gc.ActivityStateCanceled:
			// 完了/キャンセル状態はカウントしない
		}
	}))

	return summary
}

func TestStartActivity(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

	// アクティビティを作成
	comp, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)

	// アクティビティ開始
	err = StartActivity(comp, actor, world)
	assert.NoError(t, err)

	// アクティビティが登録されているかチェック
	currentActivity := worldhelper.GetActivity(world, actor)
	assert.NotNil(t, currentActivity, "Expected activity to be registered")
	assert.Equal(t, comp, currentActivity, "Expected registered activity to match started activity")

	// HasActivity のテスト
	assert.True(t, worldhelper.HasActivity(world, actor), "Expected HasActivity to return true")

	// 存在しないエンティティのテスト
	nonExistentActor := ecs.Entity(999)
	assert.False(t, worldhelper.HasActivity(world, nonExistentActor), "Expected HasActivity to return false for non-existent entity")
}

func TestMultipleActivities(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor1 := world.Manager.NewEntity()
	actor1.AddComponent(world.Components.TurnBased, &gc.TurnBased{})
	actor2 := world.Manager.NewEntity()
	actor2.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

	// 複数のアクターでアクティビティを開始
	comp1, err := NewActivity(&WaitActivity{}, 10)
	require.NoError(t, err)
	comp2, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)

	err = StartActivity(comp1, actor1, world)
	assert.NoError(t, err)

	err = StartActivity(comp2, actor2, world)
	assert.NoError(t, err)

	// 両方のアクティビティが登録されているかチェック
	assert.True(t, worldhelper.HasActivity(world, actor1), "Expected actor1 to have activity")
	assert.True(t, worldhelper.HasActivity(world, actor2), "Expected actor2 to have activity")

	// 正しいアクティビティが取得できるかチェック
	retrievedActivity1 := worldhelper.GetActivity(world, actor1)
	assert.NotNil(t, retrievedActivity1, "Expected actor1 to have activity")

	retrievedActivity2 := worldhelper.GetActivity(world, actor2)
	assert.NotNil(t, retrievedActivity2, "Expected actor2 to have activity")
}

func TestReplaceActivity(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

	// 最初のアクティビティを開始
	comp1, err := NewActivity(&WaitActivity{}, 10)
	require.NoError(t, err)
	err = StartActivity(comp1, actor, world)
	assert.NoError(t, err)

	// 最初のアクティビティが実行中であることを確認
	assert.Equal(t, gc.ActivityStateRunning, comp1.State, "Expected first activity to be running")

	// 新しいアクティビティを開始（古いものを置き換え）
	comp2, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)
	err = StartActivity(comp2, actor, world)
	assert.NoError(t, err)

	// 古いアクティビティが中断されているかチェック
	assert.Equal(t, gc.ActivityStatePaused, comp1.State, "Expected first activity to be paused after replacement")

	// 新しいアクティビティが現在のアクティビティになっているかチェック
	currentActivity := worldhelper.GetActivity(world, actor)
	assert.Equal(t, comp2, currentActivity, "Expected current activity to be the second activity")
}

func TestInterruptAndResume(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

	// アクティビティを開始
	comp, err := NewActivity(&WaitActivity{}, 10)
	require.NoError(t, err)
	err = StartActivity(comp, actor, world)
	assert.NoError(t, err)

	// アクティビティを中断
	err = InterruptActivity(actor, "テスト中断", world)
	assert.NoError(t, err)

	assert.Equal(t, gc.ActivityStatePaused, comp.State, "Expected activity to be paused after interrupt")

	// 中断されたアクティビティはアクティブではない
	assert.False(t, worldhelper.HasActivity(world, actor), "Expected HasActivity to return false for paused activity")

	// アクティビティを再開
	err = ResumeActivity(actor, world)
	assert.NoError(t, err)

	assert.Equal(t, gc.ActivityStateRunning, comp.State, "Expected activity to be running after resume")

	// 再開されたアクティビティはアクティブ
	assert.True(t, worldhelper.HasActivity(world, actor), "Expected HasActivity to return true for resumed activity")

	// 存在しないアクティビティの中断・再開テスト
	nonExistentActor := ecs.Entity(999)
	err = InterruptActivity(nonExistentActor, "テスト", world)
	assert.Error(t, err, "Expected error when interrupting non-existent activity")

	err = ResumeActivity(nonExistentActor, world)
	assert.Error(t, err, "Expected error when resuming non-existent activity")
}

func TestCancelActivity(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

	// アクティビティを開始
	comp, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)
	err = StartActivity(comp, actor, world)
	assert.NoError(t, err)

	// アクティビティをキャンセル
	CancelActivity(actor, "テストキャンセル", world)

	assert.Equal(t, gc.ActivityStateCanceled, comp.State, "Expected activity to be canceled")

	// キャンセルされたアクティビティは管理対象から削除される
	currentActivity := worldhelper.GetActivity(world, actor)
	assert.Nil(t, currentActivity, "Expected no current activity after cancel")

	// 存在しないアクティビティのキャンセル（エラーにならない）
	nonExistentActor := ecs.Entity(999)
	CancelActivity(nonExistentActor, "テスト", world) // パニックしないことを確認
}

func TestProcessTurn(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor1 := world.Manager.NewEntity()
	actor1.AddComponent(world.Components.TurnBased, &gc.TurnBased{})
	actor2 := world.Manager.NewEntity()
	actor2.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

	// 短いアクティビティと長いアクティビティを開始
	shortComp, err := NewActivity(&WaitActivity{}, 2) // 2ターンで完了
	require.NoError(t, err)
	longComp, err := NewActivity(&WaitActivity{}, 5) // 5ターンで完了
	require.NoError(t, err)

	err = StartActivity(shortComp, actor1, world)
	assert.NoError(t, err)
	err = StartActivity(longComp, actor2, world)
	assert.NoError(t, err)

	// 初期状態の確認
	summary := getActivitySummary(t, world)
	assert.Equal(t, 2, summary["total"], "Expected 2 total activities")
	assert.Equal(t, 2, summary["active"], "Expected 2 active activities")

	// 1ターン目処理
	ProcessTurn(world)

	// 両方まだ実行中
	assert.Equal(t, 1, shortComp.TurnsLeft, "Expected short activity to have 1 turn left")
	assert.Equal(t, 4, longComp.TurnsLeft, "Expected long activity to have 4 turns left")

	// 2ターン目処理
	ProcessTurn(world)

	// 短いアクティビティが完了
	assert.True(t, IsCompleted(shortComp), "Expected short activity to be completed")
	assert.Equal(t, 3, longComp.TurnsLeft, "Expected long activity to have 3 turns left")

	// 完了したアクティビティは管理対象から削除される
	assert.Nil(t, worldhelper.GetActivity(world, actor1), "Expected completed activity to be removed")
	assert.NotNil(t, worldhelper.GetActivity(world, actor2), "Expected long activity to still be present")

	// サマリーの確認
	summary = getActivitySummary(t, world)
	assert.Equal(t, 1, summary["total"], "Expected 1 total activity after completion")
}

func TestActivitySummary(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 初期状態のサマリー
	summary := getActivitySummary(t, world)
	assert.Equal(t, 0, summary["total"], "Expected 0 total activities initially")
	assert.Equal(t, 0, summary["active"], "Expected 0 active activities initially")
	assert.Equal(t, 0, summary["paused"], "Expected 0 paused activities initially")

	// アクティビティを追加
	actor1 := world.Manager.NewEntity()
	actor1.AddComponent(world.Components.TurnBased, &gc.TurnBased{})
	actor2 := world.Manager.NewEntity()
	actor2.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

	comp1, err := NewActivity(&WaitActivity{}, 10)
	require.NoError(t, err)
	comp2, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)

	err = StartActivity(comp1, actor1, world)
	assert.NoError(t, err)
	err = StartActivity(comp2, actor2, world)
	assert.NoError(t, err)

	// 1つを中断
	err = InterruptActivity(actor1, "テスト", world)
	assert.NoError(t, err)

	// サマリーの確認
	summary = getActivitySummary(t, world)
	assert.Equal(t, 2, summary["total"], "Expected 2 total activities")
	assert.Equal(t, 1, summary["active"], "Expected 1 active activity")
	assert.Equal(t, 1, summary["paused"], "Expected 1 paused activity")
}

func TestLastActivity(t *testing.T) {
	t.Parallel()

	t.Run("結果が記録される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 5, 5, "セレスティン")
		require.NoError(t, err)

		params := ActionParams{
			Actor:    player,
			Duration: 1,
			Reason:   "テスト",
		}
		_, err = Execute(&WaitActivity{}, params, world)
		require.NoError(t, err)

		result := GetLastResult(player, world)
		expected := &gc.LastActivity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateCompleted,
			Success:      true,
			Message:      "アクション完了",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("複数のアクティビティで最新の結果が保持される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// 待機
		params := ActionParams{Actor: player, Duration: 1, Reason: "待機"}
		_, err = Execute(&WaitActivity{}, params, world)
		require.NoError(t, err)

		result := GetLastResult(player, world)
		expected := &gc.LastActivity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateCompleted,
			Success:      true,
			Message:      "アクション完了",
		}
		assert.Equal(t, expected, result)

		// 移動
		params = ActionParams{Actor: player, Destination: &gc.GridElement{X: 10, Y: 9}}
		_, err = Execute(&MoveActivity{}, params, world)
		require.NoError(t, err)

		result = GetLastResult(player, world)
		expected = &gc.LastActivity{
			BehaviorName: gc.BehaviorMove,
			State:        gc.ActivityStateCompleted,
			Success:      true,
			Message:      "アクション完了",
		}
		assert.Equal(t, expected, result)
	})

	t.Run("失敗したアクティビティも記録される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 5, 5, "セレスティン")
		require.NoError(t, err)

		// 存在しないターゲットへの攻撃（失敗する）
		nonExistentEntity := ecs.Entity(9999)
		params := ActionParams{Actor: player, Target: &nonExistentEntity}
		_, _ = Execute(&AttackActivity{}, params, world)

		result := GetLastResult(player, world)
		expected := &gc.LastActivity{
			BehaviorName: gc.BehaviorAttack,
			State:        gc.ActivityStateCanceled,
			Success:      false,
			Message:      "アクティビティ検証失敗: " + ErrAttackTargetNotExists.Error(),
		}
		assert.Equal(t, expected, result)
	})
}
