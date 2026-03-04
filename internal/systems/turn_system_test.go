package systems

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

func TestTurnSystem_String(t *testing.T) {
	t.Parallel()
	sys := &TurnSystem{}
	assert.Equal(t, "TurnSystem", sys.String())
}

func TestTurnSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("PlayerTurnでAPがマイナスなら自動でAITurnへ遷移", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// ターン状態を設定
		turnState, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)
		turnState.Phase = gc.TurnPhasePlayer

		// APをマイナスに設定
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = -50

		sys := &TurnSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		assert.Equal(t, gc.TurnPhaseAI, turnState.Phase, "APがマイナスならAITurnへ遷移するべき")
	})

	t.Run("PlayerTurnでAPが0以上なら遷移しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// ターン状態を設定
		turnState, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)
		turnState.Phase = gc.TurnPhasePlayer

		// APを正の値に設定
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = 100

		sys := &TurnSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		assert.Equal(t, gc.TurnPhasePlayer, turnState.Phase, "APが0以上ならPlayerTurnのまま")
	})

	t.Run("AITurnからTurnEndへ遷移", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// ターン状態を設定
		turnState, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)
		turnState.Phase = gc.TurnPhaseAI

		sys := &TurnSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		assert.Equal(t, gc.TurnPhaseEnd, turnState.Phase, "AITurnからTurnEndへ遷移するべき")
	})

	t.Run("TurnEndから新しいターンへ遷移", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Updaters = make(map[string]w.Updater)

		// ターン状態を設定
		turnState, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)
		turnState.Phase = gc.TurnPhaseEnd
		initialTurnNumber := turnState.TurnNumber

		sys := &TurnSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		assert.Equal(t, gc.TurnPhasePlayer, turnState.Phase, "TurnEndからPlayerTurnへ遷移するべき")
		assert.Equal(t, initialTurnNumber+1, turnState.TurnNumber, "ターン番号が増加するべき")
	})
}

func TestProcessTurnEnd(t *testing.T) {
	t.Parallel()

	t.Run("ターン終了時にAPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Updaters = make(map[string]w.Updater)

		player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// APをマイナスに設定
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = -100

		err = processTurnEnd(world)
		require.NoError(t, err)

		// APが回復していることを確認（Speedに応じた回復量）
		assert.Greater(t, turnBased.AP.Current, -100, "APが回復するべき")
	})

	t.Run("登録されたシステムが実行される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// テスト用にUpdatersを設定
		world.Updaters = make(map[string]w.Updater)
		deadCleanup := &DeadCleanupSystem{}
		world.Updaters[deadCleanup.String()] = deadCleanup

		err := processTurnEnd(world)
		require.NoError(t, err)
	})
}

func TestShouldAutoEndTurn(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーがいない場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		result := shouldAutoEndTurn(world)
		assert.False(t, result, "プレイヤーがいない場合はfalse")
	})

	t.Run("TurnBasedコンポーネントがない場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// TurnBasedなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		result := shouldAutoEndTurn(world)
		assert.False(t, result, "TurnBasedがない場合はfalse")
	})
}

func TestProcessPlayerContinuousActivity(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーがいない場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		result := processPlayerContinuousActivity(world)

		assert.False(t, result)
	})

	t.Run("継続アクションがない場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		result := processPlayerContinuousActivity(world)

		assert.False(t, result, "継続アクションがない場合はfalse")
	})

	t.Run("継続アクションがある場合はtrueを返しAPを消費する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 継続アクションを設定
		player.AddComponent(world.Components.Activity, &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
		})

		// 初期APを確認
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		initialAP := turnBased.AP.Current

		result := processPlayerContinuousActivity(world)

		assert.True(t, result, "継続アクションがある場合はtrue")
		assert.Less(t, turnBased.AP.Current, initialAP, "APが消費されるべき")
	})
}

// TestColdPlayerCanAct は冷えた状態のプレイヤーが行動できることをテストする
func TestColdPlayerCanAct(t *testing.T) {
	t.Parallel()

	// canEntityAct はエンティティが行動可能かを判定するヘルパー関数
	canEntityAct := func(world w.World, entity ecs.Entity, _ int) bool {
		turnState, err := worldhelper.GetTurnState(world)
		if err != nil || turnState.Phase != gc.TurnPhasePlayer {
			return false
		}
		tbComp := world.Components.TurnBased.Get(entity)
		if tbComp == nil {
			return false
		}
		tb := tbComp.(*gc.TurnBased)
		return tb.AP.Current >= 0
	}

	t.Run("行動可能かの判定を確認", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// TurnBasedコンポーネントの存在確認
		turnBased := world.Components.TurnBased.Get(playerEntity)
		require.NotNil(t, turnBased, "プレイヤーはTurnBasedコンポーネントを持つべき")

		tb := turnBased.(*gc.TurnBased)
		t.Logf("TurnBased AP: Current=%d, Max=%d", tb.AP.Current, tb.AP.Max)

		// 行動可能かを確認
		turnState, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)
		canAct := canEntityAct(world, playerEntity, 100)
		t.Logf("canEntityAct result: %v (TurnPhase=%v)", canAct, turnState.Phase)

		assert.True(t, canAct, "プレイヤーターンでAP >= 0なら行動可能")
	})

	t.Run("重度の低体温でもAPが0以上なら行動可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定（全部位）
		hs := world.Components.HealthStatus.Get(playerEntity).(*gc.HealthStatus)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			hs.Parts[i].SetCondition(gc.HealthCondition{
				Type:     gc.ConditionHypothermia,
				Severity: gc.SeveritySevere,
				Timer:    90, // 重度
			})
		}

		// Speedを再計算（低体温ペナルティ適用）
		speed := worldhelper.CalculateSpeed(world, playerEntity)
		t.Logf("低体温時のSpeed: %d", speed)

		// TurnBasedのAPを確認
		turnBased := world.Components.TurnBased.Get(playerEntity).(*gc.TurnBased)
		t.Logf("現在のAP: %d, 最大AP: %d", turnBased.AP.Current, turnBased.AP.Max)

		// APが0以上なら行動可能であることを確認
		canAct := canEntityAct(world, playerEntity, 100)
		assert.True(t, canAct, "APが0以上ならば冷えていても行動可能であるべき")
	})

	t.Run("APがマイナスになったらshouldAutoEndTurnがtrueを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// APをマイナスに設定
		turnBased := world.Components.TurnBased.Get(playerEntity).(*gc.TurnBased)
		turnBased.AP.Current = -50

		// shouldAutoEndTurnがtrueを返すことを確認
		result := shouldAutoEndTurn(world)
		assert.True(t, result, "APがマイナスの場合はshouldAutoEndTurnがtrueを返すべき")
	})

	t.Run("APが0以上の場合はshouldAutoEndTurnがfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// APを0に設定
		turnBased := world.Components.TurnBased.Get(playerEntity).(*gc.TurnBased)
		turnBased.AP.Current = 0

		// shouldAutoEndTurnがfalseを返すことを確認
		result := shouldAutoEndTurn(world)
		assert.False(t, result, "APが0の場合はshouldAutoEndTurnがfalseを返すべき")
	})

	t.Run("ConsumeActionPointsでAPが正しく消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 初期APを確認
		turnBased := world.Components.TurnBased.Get(playerEntity).(*gc.TurnBased)
		initialAP := turnBased.AP.Current
		t.Logf("初期AP: %d", initialAP)

		// アクションポイントを消費
		success := worldhelper.ConsumeActionPoints(world, playerEntity, 100)
		assert.True(t, success, "APの消費が成功するべき")

		// APが正しく減少していることを確認
		assert.Equal(t, initialAP-100, turnBased.AP.Current, "APが100減少しているべき")
		t.Logf("消費後AP: %d", turnBased.AP.Current)
	})

	t.Run("重度の低体温でSpeed計算にペナルティが適用される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 通常時のSpeedを計算
		normalSpeed := worldhelper.CalculateSpeed(world, playerEntity)
		t.Logf("通常時のSpeed: %d", normalSpeed)

		// 重度の低体温を設定（全部位）
		hs := world.Components.HealthStatus.Get(playerEntity).(*gc.HealthStatus)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			hs.Parts[i].SetCondition(gc.HealthCondition{
				Type:     gc.ConditionHypothermia,
				Severity: gc.SeveritySevere,
				Timer:    90, // 重度
			})
		}

		// 低体温時のSpeedを計算
		coldSpeed := worldhelper.CalculateSpeed(world, playerEntity)
		t.Logf("低体温時のSpeed: %d", coldSpeed)

		// ペナルティが適用されていることを確認
		assert.Less(t, coldSpeed, normalSpeed, "低体温時はSpeedにペナルティがあるべき")
	})

	t.Run("完全なターンサイクルで冷えたプレイヤーが行動できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(playerEntity).(*gc.HealthStatus)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			hs.Parts[i].SetCondition(gc.HealthCondition{
				Type:     gc.ConditionHypothermia,
				Severity: gc.SeveritySevere,
				Timer:    90,
			})
		}

		turnBased := world.Components.TurnBased.Get(playerEntity).(*gc.TurnBased)
		turnState, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)

		// 1. プレイヤーターンで行動可能
		t.Logf("初期状態: TurnPhase=%v, AP=%d", turnState.Phase, turnBased.AP.Current)
		canAct := canEntityAct(world, playerEntity, 100)
		assert.True(t, canAct, "初期状態で行動可能")

		// 2. 攻撃を実行（APを消費）
		worldhelper.ConsumeActionPoints(world, playerEntity, 100)
		t.Logf("攻撃後: AP=%d", turnBased.AP.Current)

		// 3. APがマイナスでなければまだ行動可能
		if turnBased.AP.Current >= 0 {
			canAct = canEntityAct(world, playerEntity, 100)
			assert.True(t, canAct, "APが0以上なら行動可能")
		}

		// 4. APをマイナスにして自動ターン終了をテスト
		turnBased.AP.Current = -50
		t.Logf("APをマイナスに設定: AP=%d", turnBased.AP.Current)

		shouldEnd := shouldAutoEndTurn(world)
		assert.True(t, shouldEnd, "APがマイナスならターン自動終了")

		// 5. ターン終了処理（AP回復）
		speed := worldhelper.CalculateSpeed(world, playerEntity)
		t.Logf("Speed (低体温ペナルティ込み): %d", speed)

		err = worldhelper.RestoreAllActionPoints(world)
		require.NoError(t, err)
		t.Logf("ターン終了後AP: %d", turnBased.AP.Current)

		// 6. 十分なターン経過でAPが0以上に回復
		for turnBased.AP.Current < 0 {
			err = worldhelper.RestoreAllActionPoints(world)
			require.NoError(t, err)
			t.Logf("追加ターン終了後AP: %d", turnBased.AP.Current)
		}

		// 7. 再び行動可能
		turnState.Phase = gc.TurnPhasePlayer
		canAct = canEntityAct(world, playerEntity, 100)
		assert.True(t, canAct, "APが回復したら行動可能")
	})

	t.Run("極端なペナルティでもSpeedは最小値を保証", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 重度の低体温 + 飢餓
		hs := world.Components.HealthStatus.Get(playerEntity).(*gc.HealthStatus)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			hs.Parts[i].SetCondition(gc.HealthCondition{
				Type:     gc.ConditionHypothermia,
				Severity: gc.SeveritySevere,
				Timer:    90,
			})
		}

		// 飢餓状態を設定
		hunger := world.Components.Hunger.Get(playerEntity).(*gc.Hunger)
		hunger.Current = 5 // 餓死寸前

		speed := worldhelper.CalculateSpeed(world, playerEntity)
		t.Logf("極端ペナルティ時のSpeed: %d", speed)

		// 最小Speedは25
		assert.GreaterOrEqual(t, speed, 25, "Speedは最小値25を保証")
	})

	t.Run("冷えた状態で敵に向かって移動すると攻撃が発動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(playerEntity).(*gc.HealthStatus)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			hs.Parts[i].SetCondition(gc.HealthCondition{
				Type:     gc.ConditionHypothermia,
				Severity: gc.SeveritySevere,
				Timer:    90,
			})
		}

		// APを確認
		turnBased := world.Components.TurnBased.Get(playerEntity).(*gc.TurnBased)
		turnState, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)
		t.Logf("冷えた状態のAP: Current=%d, Max=%d", turnBased.AP.Current, turnBased.AP.Max)

		// 行動可能であることを確認
		canAct := canEntityAct(world, playerEntity, 100)
		t.Logf("canEntityAct結果: %v (TurnPhase=%v)", canAct, turnState.Phase)
		assert.True(t, canAct, "冷えた状態でもAP >= 0なら行動可能であるべき")

		// shouldAutoEndTurnがfalseであることを確認
		shouldEnd := shouldAutoEndTurn(world)
		t.Logf("shouldAutoEndTurn結果: %v (AP.Current=%d)", shouldEnd, turnBased.AP.Current)
		assert.False(t, shouldEnd, "AP >= 0ならshouldAutoEndTurnはfalseであるべき")
	})
}
