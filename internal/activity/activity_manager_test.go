package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getActivitySummary はテスト用にアクティビティの要約情報を取得する
func getActivitySummary(t *testing.T, world w.World) map[string]int {
	t.Helper()
	summary := map[string]int{
		"total":  0,
		"active": 0,
		"paused": 0,
	}

	activityQuery := ecs.NewFilter1[gc.Activity](world.ECS).Query()
	for activityQuery.Next() {
		entity := activityQuery.Entity()
		comp := world.Components.Activity.Get(entity)
		summary["total"]++
		switch comp.State {
		case gc.ActivityStateRunning:
			summary["active"]++
		case gc.ActivityStatePaused:
			summary["paused"]++
		case gc.ActivityStateCompleted, gc.ActivityStateCanceled:
			// 完了/キャンセル状態はカウントしない
		}
	}

	return summary
}

func TestStartActivity(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor, &gc.TurnBased{})

	// アクティビティを作成
	comp, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)

	// アクティビティ開始
	err = StartActivity(comp, actor, world)
	require.NoError(t, err)

	// アクティビティが登録されているかチェック
	currentActivity := query.GetActivity(world, actor)
	assert.NotNil(t, currentActivity, "Expected activity to be registered")
	assert.Equal(t, comp, currentActivity, "Expected registered activity to match started activity")

	// HasActivity のテスト
	assert.True(t, query.HasActivity(world, actor), "Expected HasActivity to return true")

	// 存在しないエンティティのテスト
	nonExistentActor := consts.InvalidEntity
	assert.False(t, query.HasActivity(world, nonExistentActor), "Expected HasActivity to return false for non-existent entity")
}

func TestMultipleActivities(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor1 := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor1, &gc.TurnBased{})
	actor2 := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor2, &gc.TurnBased{})

	// 複数のアクターでアクティビティを開始
	comp1, err := NewActivity(&WaitActivity{}, 10)
	require.NoError(t, err)
	comp2, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)

	err = StartActivity(comp1, actor1, world)
	require.NoError(t, err)

	err = StartActivity(comp2, actor2, world)
	require.NoError(t, err)

	// 両方のアクティビティが登録されているかチェック
	assert.True(t, query.HasActivity(world, actor1), "Expected actor1 to have activity")
	assert.True(t, query.HasActivity(world, actor2), "Expected actor2 to have activity")

	// 正しいアクティビティが取得できるかチェック
	retrievedActivity1 := query.GetActivity(world, actor1)
	assert.NotNil(t, retrievedActivity1, "Expected actor1 to have activity")

	retrievedActivity2 := query.GetActivity(world, actor2)
	assert.NotNil(t, retrievedActivity2, "Expected actor2 to have activity")
}

func TestReplaceActivity(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor, &gc.TurnBased{})

	// 最初のアクティビティを開始
	comp1, err := NewActivity(&WaitActivity{}, 10)
	require.NoError(t, err)
	err = StartActivity(comp1, actor, world)
	require.NoError(t, err)

	// 最初のアクティビティが実行中であることを確認
	assert.Equal(t, gc.ActivityStateRunning, comp1.State, "Expected first activity to be running")

	// 新しいアクティビティを開始（古いものを置き換え）
	comp2, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)
	err = StartActivity(comp2, actor, world)
	require.NoError(t, err)

	// 置き換え後は現在のアクティビティが2つ目になっている。
	// Arkは値で格納しアクティビティスタックを持たないため、中断された1つ目は破棄される
	currentActivity := query.GetActivity(world, actor)
	require.NotNil(t, currentActivity)
	assert.Equal(t, gc.ActivityStateRunning, currentActivity.State, "Expected replaced activity to be running")
	assert.Equal(t, comp2.TurnsTotal, currentActivity.TurnsTotal, "Expected current activity to be the second activity")
}

func TestInterruptAndResume(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor, &gc.TurnBased{})

	// アクティビティを開始
	comp, err := NewActivity(&WaitActivity{}, 10)
	require.NoError(t, err)
	err = StartActivity(comp, actor, world)
	require.NoError(t, err)

	// アクティビティを中断
	err = InterruptActivity(actor, "テスト中断", world)
	require.NoError(t, err)

	assert.Equal(t, gc.ActivityStatePaused, query.GetActivity(world, actor).State, "Expected activity to be paused after interrupt")

	// 中断されたアクティビティはアクティブではない
	assert.False(t, query.HasActivity(world, actor), "Expected HasActivity to return false for paused activity")

	// アクティビティを再開
	err = ResumeActivity(actor, world)
	require.NoError(t, err)

	assert.Equal(t, gc.ActivityStateRunning, query.GetActivity(world, actor).State, "Expected activity to be running after resume")

	// 再開されたアクティビティはアクティブ
	assert.True(t, query.HasActivity(world, actor), "Expected HasActivity to return true for resumed activity")

	// 存在しないアクティビティの中断・再開テスト
	nonExistentActor := consts.InvalidEntity
	err = InterruptActivity(nonExistentActor, "テスト", world)
	require.Error(t, err, "Expected error when interrupting non-existent activity")

	err = ResumeActivity(nonExistentActor, world)
	assert.Error(t, err, "Expected error when resuming non-existent activity")
}

func TestCancelActivity(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor, &gc.TurnBased{})

	// アクティビティを開始
	comp, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)
	err = StartActivity(comp, actor, world)
	require.NoError(t, err)

	// アクティビティをキャンセル
	CancelActivity(actor, "テストキャンセル", world)

	// キャンセル時は削除されるため結果コンポーネントで状態を確認する
	assert.Equal(t, gc.ActivityStateCanceled, GetLastResult(actor, world).State, "Expected activity to be canceled")

	// キャンセルされたアクティビティは管理対象から削除される
	currentActivity := query.GetActivity(world, actor)
	assert.Nil(t, currentActivity, "Expected no current activity after cancel")

	// 存在しないアクティビティのキャンセル（エラーにならない）
	nonExistentActor := consts.InvalidEntity
	CancelActivity(nonExistentActor, "テスト", world) // パニックしないことを確認
}

func TestProcessTurn(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	actor1 := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor1, &gc.TurnBased{})
	actor2 := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor2, &gc.TurnBased{})

	// 短いアクティビティと長いアクティビティを開始
	shortComp, err := NewActivity(&WaitActivity{}, 2) // 2ターンで完了
	require.NoError(t, err)
	longComp, err := NewActivity(&WaitActivity{}, 5) // 5ターンで完了
	require.NoError(t, err)

	err = StartActivity(shortComp, actor1, world)
	require.NoError(t, err)
	err = StartActivity(longComp, actor2, world)
	require.NoError(t, err)

	// 初期状態の確認
	summary := getActivitySummary(t, world)
	assert.Equal(t, 2, summary["total"], "Expected 2 total activities")
	assert.Equal(t, 2, summary["active"], "Expected 2 active activities")

	// 1ターン目処理
	ProcessTurn(world)

	// 両方まだ実行中。Arkは値で格納するため格納側を取り直して検証する
	assert.Equal(t, 1, query.GetActivity(world, actor1).TurnsLeft, "Expected short activity to have 1 turn left")
	assert.Equal(t, 4, query.GetActivity(world, actor2).TurnsLeft, "Expected long activity to have 4 turns left")

	// 2ターン目処理
	ProcessTurn(world)

	// 短いアクティビティが完了。完了時は削除されるため結果コンポーネントで確認する
	assert.Equal(t, gc.ActivityStateCompleted, GetLastResult(actor1, world).State, "Expected short activity to be completed")
	assert.Equal(t, 3, query.GetActivity(world, actor2).TurnsLeft, "Expected long activity to have 3 turns left")

	// 完了したアクティビティは管理対象から削除される
	assert.Nil(t, query.GetActivity(world, actor1), "Expected completed activity to be removed")
	assert.NotNil(t, query.GetActivity(world, actor2), "Expected long activity to still be present")

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
	actor1 := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor1, &gc.TurnBased{})
	actor2 := world.ECS.NewEntity()
	world.Components.TurnBased.Add(actor2, &gc.TurnBased{})

	comp1, err := NewActivity(&WaitActivity{}, 10)
	require.NoError(t, err)
	comp2, err := NewActivity(&WaitActivity{}, 5)
	require.NoError(t, err)

	err = StartActivity(comp1, actor1, world)
	require.NoError(t, err)
	err = StartActivity(comp2, actor2, world)
	require.NoError(t, err)

	// 1つを中断
	err = InterruptActivity(actor1, "テスト", world)
	require.NoError(t, err)

	// サマリーの確認
	summary = getActivitySummary(t, world)
	assert.Equal(t, 2, summary["total"], "Expected 2 total activities")
	assert.Equal(t, 1, summary["active"], "Expected 1 active activity")
	assert.Equal(t, 1, summary["paused"], "Expected 1 paused activity")
}

func TestGetPassCostAt(t *testing.T) {
	t.Parallel()

	t.Run("PassCostがないタイルでは0を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		cost := getPassCostAt(world, 5, 5)
		assert.Equal(t, 0, cost)
	})

	t.Run("PassCostがあるタイルではValue値を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		prop := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop, &gc.GridElement{X: 5, Y: 5})
		world.Components.PassCost.Add(prop, &gc.PassCost{Value: 50})

		cost := getPassCostAt(world, 5, 5)
		assert.Equal(t, 50, cost)
	})

	t.Run("同一タイルに複数のPassCostがある場合は合算する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		prop1 := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop1, &gc.GridElement{X: 5, Y: 5})
		world.Components.PassCost.Add(prop1, &gc.PassCost{Value: 30})

		prop2 := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop2, &gc.GridElement{X: 5, Y: 5})
		world.Components.PassCost.Add(prop2, &gc.PassCost{Value: 20})

		cost := getPassCostAt(world, 5, 5)
		assert.Equal(t, 50, cost)
	})
}

func TestConsumePassCostWithPassCost(t *testing.T) {
	t.Parallel()

	t.Run("PassCostがある移動先ではAPコストが増加する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 通常移動のAP消費を記録する。
		// ExecuteはArchetypeを変える構造変更を伴うため、TurnBasedは都度取り直す
		apBefore := world.Components.TurnBased.Get(player).AP.Current

		_, err = Execute(&MoveActivity{Destination: gc.GridElement{X: 11, Y: 10}}, player, world)
		require.NoError(t, err)

		normalCost := apBefore - world.Components.TurnBased.Get(player).AP.Current

		// APをリセットしてPassCostありの移動をテスト
		world.Components.TurnBased.Get(player).AP.Current = apBefore

		// 移動先にPassCostを持つPropを配置
		prop := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop, &gc.GridElement{X: 12, Y: 10})
		world.Components.PassCost.Add(prop, &gc.PassCost{Value: 50})

		_, err = Execute(&MoveActivity{Destination: gc.GridElement{X: 12, Y: 10}}, player, world)
		require.NoError(t, err)

		modCost := apBefore - world.Components.TurnBased.Get(player).AP.Current
		// PassCost 50加算: normalCost + 50
		assert.Equal(t, normalCost+50, modCost)
	})
}

func TestLastActivity(t *testing.T) {
	t.Parallel()

	t.Run("結果が記録される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		_, err = Execute(&WaitActivity{Duration: 1, Reason: "テスト"}, player, world)
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 待機
		_, err = Execute(&WaitActivity{Duration: 1, Reason: "待機"}, player, world)
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
		_, err = Execute(&MoveActivity{Destination: gc.GridElement{X: 10, Y: 9}}, player, world)
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

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 存在しないターゲットへの攻撃（失敗する）
		nonExistentEntity := consts.InvalidEntity
		_, _ = Execute(&AttackActivity{Target: nonExistentEntity}, player, world)

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
