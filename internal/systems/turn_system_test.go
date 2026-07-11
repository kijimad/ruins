package systems

import (
	"math/rand/v2"
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

func TestTurnSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("PlayerTurnでAPがマイナスなら自動でAITurnへ遷移", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// ターン状態を設定
		turnState := query.GetTurnState(world)

		turnState.Phase = gc.TurnPhasePlayer

		// APをマイナスに設定
		turnBased := world.Components.TurnBased.Get(player)
		turnBased.AP.Current = -50

		sys := &TurnSystem{}
		err = sys.Update(world)
		require.NoError(t, err)

		assert.Equal(t, gc.TurnPhaseAI, turnState.Phase, "APがマイナスならAITurnへ遷移するべき")
	})

	t.Run("PlayerTurnでAPが0以上なら遷移しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// ターン状態を設定
		turnState := query.GetTurnState(world)

		turnState.Phase = gc.TurnPhasePlayer

		// APを正の値に設定
		turnBased := world.Components.TurnBased.Get(player)
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
		turnState := query.GetTurnState(world)

		turnState.Phase = gc.TurnPhaseAI

		sys := &TurnSystem{}
		err := sys.Update(world)
		require.NoError(t, err)

		assert.Equal(t, gc.TurnPhaseEnd, turnState.Phase, "AITurnからTurnEndへ遷移するべき")
	})

	t.Run("TurnEndから新しいターンへ遷移", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Updaters = make(map[string]w.Updater)

		// ターン状態を設定
		turnState := query.GetTurnState(world)

		turnState.Phase = gc.TurnPhaseEnd
		initialTurnNumber := turnState.TurnNumber

		sys := &TurnSystem{}
		err := sys.Update(world)
		require.NoError(t, err)

		assert.Equal(t, gc.TurnPhasePlayer, turnState.Phase, "TurnEndからPlayerTurnへ遷移するべき")
		assert.Equal(t, initialTurnNumber+1, turnState.TurnNumber, "ターン番号が増加するべき")
	})
}

// TestDeadCleanupBeforeTurnSystem はDungeonState.Updateと同じ実行順序
// （DeadCleanupSystem → TurnSystem）で、複数回行動時にDeadが即座に消えることを検証する
func TestDeadCleanupBeforeTurnSystem(t *testing.T) {
	t.Parallel()

	// runFrame はDungeonState.Updateのシステム実行順序を模擬する
	runFrame := func(world w.World) error {
		deadSys := &DeadCleanupSystem{}
		if err := deadSys.Update(world); err != nil {
			return err
		}
		turnSys := &TurnSystem{}
		return turnSys.Update(world)
	}

	t.Run("APが余っているPlayerPhase中でもDeadエンティティが削除される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player)
		turnBased.AP.Current = 200

		turnState := query.GetTurnState(world)

		turnState.Phase = gc.TurnPhasePlayer

		// 敵を作成してDeadコンポーネントを付与（射撃で倒された状態を模擬）
		enemy := world.ECS.NewEntity()
		world.Components.Name.Add(enemy, &gc.Name{Name: "スライム"})
		world.Components.Dead.Add(enemy, &gc.Dead{})

		err = runFrame(world)
		require.NoError(t, err)

		assert.Equal(t, gc.TurnPhasePlayer, turnState.Phase)
		assert.Equal(t, 200, turnBased.AP.Current, "APは消費されていない")
		assert.False(t, world.ECS.Alive(enemy),
			"PlayerPhase中でもDeadエンティティは削除されるべき")
	})

	t.Run("複数回行動の間にDeadエンティティが消える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player)
		turnBased.AP.Current = 300

		turnState := query.GetTurnState(world)

		turnState.Phase = gc.TurnPhasePlayer

		// 1回目の行動: 敵1を倒す
		enemy1 := world.ECS.NewEntity()
		world.Components.Name.Add(enemy1, &gc.Name{Name: "スライム1"})
		world.Components.Dead.Add(enemy1, &gc.Dead{})

		query.ConsumeActionPoints(world, player, 100) // AP: 300 -> 200

		err = runFrame(world)
		require.NoError(t, err)

		assert.False(t, world.ECS.Alive(enemy1),
			"1回目の行動後にDeadエンティティが削除されるべき")
		assert.Equal(t, gc.TurnPhasePlayer, turnState.Phase,
			"APが残っているのでPlayerPhaseのまま")

		// 2回目の行動: 敵2を倒す
		enemy2 := world.ECS.NewEntity()
		world.Components.Name.Add(enemy2, &gc.Name{Name: "スライム2"})
		world.Components.Dead.Add(enemy2, &gc.Dead{})

		query.ConsumeActionPoints(world, player, 100) // AP: 200 -> 100

		err = runFrame(world)
		require.NoError(t, err)

		assert.False(t, world.ECS.Alive(enemy2),
			"2回目の行動後にDeadエンティティが削除されるべき")
		assert.Equal(t, gc.TurnPhasePlayer, turnState.Phase,
			"APが残っているのでPlayerPhaseのまま")
	})
}

func TestProcessTurnEnd(t *testing.T) {
	t.Parallel()

	t.Run("ターン終了時にAPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Updaters = make(map[string]w.Updater)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// APをマイナスに設定
		turnBased := world.Components.TurnBased.Get(player)
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
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

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

		_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		result := processPlayerContinuousActivity(world)

		assert.False(t, result, "継続アクションがない場合はfalse")
	})

	t.Run("継続アクションがある場合はtrueを返しAPを消費する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 継続アクションを設定
		world.Components.Activity.Add(player, &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
		})

		// 初期APを確認
		turnBased := world.Components.TurnBased.Get(player)
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
		turnState := query.GetTurnState(world)
		if turnState == nil || turnState.Phase != gc.TurnPhasePlayer {
			return false
		}
		tbComp := world.Components.TurnBased.Get(entity)
		if tbComp == nil {
			return false
		}
		return tbComp.AP.Current >= 0
	}

	t.Run("行動可能かの判定を確認", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// TurnBasedコンポーネントの存在確認
		turnBased := world.Components.TurnBased.Get(playerEntity)
		require.NotNil(t, turnBased, "プレイヤーはTurnBasedコンポーネントを持つべき")

		tb := turnBased
		t.Logf("TurnBased AP: Current=%d, Max=%d", tb.AP.Current, tb.AP.Max)

		// 行動可能かを確認
		turnState := query.GetTurnState(world)
		canAct := canEntityAct(world, playerEntity, 100)
		t.Logf("canEntityAct result: %v (TurnPhase=%v)", canAct, turnState.Phase)

		assert.True(t, canAct, "プレイヤーターンでAP >= 0なら行動可能")
	})

	t.Run("重度の低体温でもAPが0以上なら行動可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(playerEntity)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		// Speedを再計算（低体温ペナルティ適用）
		speed := query.CalculateSpeed(world, playerEntity)
		t.Logf("低体温時のSpeed: %d", speed)

		// TurnBasedのAPを確認
		turnBased := world.Components.TurnBased.Get(playerEntity)
		t.Logf("現在のAP: %d, 最大AP: %d", turnBased.AP.Current, turnBased.AP.Max)

		// APが0以上なら行動可能であることを確認
		canAct := canEntityAct(world, playerEntity, 100)
		assert.True(t, canAct, "APが0以上ならば冷えていても行動可能であるべき")
	})

	t.Run("APがマイナスになったらshouldAutoEndTurnがtrueを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// APをマイナスに設定
		turnBased := world.Components.TurnBased.Get(playerEntity)
		turnBased.AP.Current = -50

		// shouldAutoEndTurnがtrueを返すことを確認
		result := shouldAutoEndTurn(world)
		assert.True(t, result, "APがマイナスの場合はshouldAutoEndTurnがtrueを返すべき")
	})

	t.Run("APが0以上の場合はshouldAutoEndTurnがfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// APを0に設定
		turnBased := world.Components.TurnBased.Get(playerEntity)
		turnBased.AP.Current = 0

		// shouldAutoEndTurnがfalseを返すことを確認
		result := shouldAutoEndTurn(world)
		assert.False(t, result, "APが0の場合はshouldAutoEndTurnがfalseを返すべき")
	})

	t.Run("ConsumeActionPointsでAPが正しく消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 初期APを確認
		turnBased := world.Components.TurnBased.Get(playerEntity)
		initialAP := turnBased.AP.Current
		t.Logf("初期AP: %d", initialAP)

		// アクションポイントを消費
		success := query.ConsumeActionPoints(world, playerEntity, 100)
		assert.True(t, success, "APの消費が成功するべき")

		// APが正しく減少していることを確認
		assert.Equal(t, initialAP-100, turnBased.AP.Current, "APが100減少しているべき")
		t.Logf("消費後AP: %d", turnBased.AP.Current)
	})

	t.Run("重度の低体温でSpeed計算にペナルティが適用される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 通常時のSpeedを計算
		normalSpeed := query.CalculateSpeed(world, playerEntity)
		t.Logf("通常時のSpeed: %d", normalSpeed)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(playerEntity)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		// Effectsコンポーネントに低体温ペナルティを反映する
		skills := world.Components.Skills.Get(playerEntity)
		abils := world.Components.Abilities.Get(playerEntity)
		effects := gc.RecalculateCharModifiers(skills, abils, hs)
		require.NoError(t, gc.Upsert(world.ECS, world.Components.CharModifiers, playerEntity, effects))

		// 低体温時のSpeedを計算
		coldSpeed := query.CalculateSpeed(world, playerEntity)
		t.Logf("低体温時のSpeed: %d", coldSpeed)

		// ペナルティが適用されていることを確認
		assert.Less(t, coldSpeed, normalSpeed, "低体温時はSpeedにペナルティがあるべき")
	})

	t.Run("完全なターンサイクルで冷えたプレイヤーが行動できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(playerEntity)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		turnBased := world.Components.TurnBased.Get(playerEntity)
		turnState := query.GetTurnState(world)

		// 1. プレイヤーターンで行動可能
		t.Logf("初期状態: TurnPhase=%v, AP=%d", turnState.Phase, turnBased.AP.Current)
		canAct := canEntityAct(world, playerEntity, 100)
		assert.True(t, canAct, "初期状態で行動可能")

		// 2. 攻撃を実行（APを消費）
		query.ConsumeActionPoints(world, playerEntity, 100)
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
		speed := query.CalculateSpeed(world, playerEntity)
		t.Logf("Speed (低体温ペナルティ込み): %d", speed)

		err = query.RestoreAllActionPoints(world)
		require.NoError(t, err)
		t.Logf("ターン終了後AP: %d", turnBased.AP.Current)

		// 6. 十分なターン経過でAPが0以上に回復
		for turnBased.AP.Current < 0 {
			err = query.RestoreAllActionPoints(world)
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
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 重度の低体温 + 飢餓
		hs := world.Components.HealthStatus.Get(playerEntity)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		// 飢餓状態を設定
		hunger := world.Components.Hunger.Get(playerEntity)
		hunger.Current = 5 // 餓死寸前

		speed := query.CalculateSpeed(world, playerEntity)
		t.Logf("極端ペナルティ時のSpeed: %d", speed)

		// 最小Speedは25
		assert.GreaterOrEqual(t, speed, 25, "Speedは最小値25を保証")
	})

	t.Run("冷えた状態で敵に向かって移動すると攻撃が発動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを生成
		playerEntity, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(playerEntity)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		// APを確認
		turnBased := world.Components.TurnBased.Get(playerEntity)
		turnState := query.GetTurnState(world)
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

// TestAIEntityActuallyMoves はAIエンティティが実際に移動することを検証する統合テスト
func TestAIEntityActuallyMoves(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Updaters = make(map[string]w.Updater)

	// プレイヤーを配置（AI処理で必要）
	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// AIエンティティを手動で作成（Driving状態で即座に移動するように設定）
	enemyX, enemyY := 20, 20
	enemy := world.ECS.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "テスト敵"})
	world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemyData{})
	world.Components.GridElement.Add(enemy, &gc.GridElement{X: consts.Tile(enemyX), Y: consts.Tile(enemyY)})
	world.Components.SoloAI.Add(enemy, &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		Movement:              gc.SoloRandom,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		ViewDistance:          5,
	})
	world.Components.TurnBased.Add(enemy, &gc.TurnBased{
		AP:    gc.IntPool{Current: 200, Max: 200},
		Speed: 100,
	})

	// AIターンを複数回実行して移動を確認
	// planRandomMoveActionは30%待機なので、十分な回数試行する
	moved := false
	for turn := range 50 {
		// AP回復
		tb := world.Components.TurnBased.Get(enemy)
		tb.AP.Current = 200

		turnState := query.GetTurnState(world)
		turnState.Phase = gc.TurnPhaseAI
		turnState.TurnNumber = turn + 1

		err := processAITurn(world)
		require.NoError(t, err)

		grid := world.Components.GridElement.Get(enemy)
		if int(grid.X) != enemyX || int(grid.Y) != enemyY {
			moved = true
			t.Logf("AIエンティティが移動した: (%d,%d) → (%d,%d) at turn %d", enemyX, enemyY, grid.X, grid.Y, turn+1)
			break
		}
	}

	assert.True(t, moved, "AIエンティティは50ターン以内に移動するべき")
}

// TestSpawnedEnemyMoves はSpawnEnemyで生成された敵が実際に移動することを検証する
func TestSpawnedEnemyMoves(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Updaters = make(map[string]w.Updater)

	// 苔亀のwander移動は80%待機の確率的挙動のため、シードを固定してflaky化を防ぐ
	world.Config.RNG = rand.New(rand.NewPCG(1, 0))

	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// SpawnEnemyで実際の敵を生成
	enemy, err := lifecycle.SpawnEnemy(world, 20, 20, "苔亀")
	require.NoError(t, err)

	initialGrid := world.Components.GridElement.Get(enemy)
	initialX, initialY := int(initialGrid.X), int(initialGrid.Y)
	t.Logf("初期位置: (%d,%d)", initialX, initialY)

	// AI状態を確認
	solo := world.Components.SoloAI.Get(enemy)
	t.Logf("初期AI: SubState=%s, StartTurn=%d, Duration=%d",
		solo.SubState, solo.StartSubStateTurn, solo.DurationSubStateTurns)

	// Waiting期間を飛ばしてDriving状態に設定
	solo.SubState = gc.AIStateDriving
	solo.DurationSubStateTurns = 100

	moved := false
	for turn := range 50 {
		// AP回復
		tb := world.Components.TurnBased.Get(enemy)
		tb.AP.Current = tb.AP.Max

		turnState := query.GetTurnState(world)
		turnState.Phase = gc.TurnPhaseAI
		turnState.TurnNumber = turn + 1

		err := processAITurn(world)
		require.NoError(t, err)

		grid := world.Components.GridElement.Get(enemy)
		if int(grid.X) != initialX || int(grid.Y) != initialY {
			moved = true
			t.Logf("SpawnEnemyの敵が移動した: (%d,%d) → (%d,%d) at turn %d",
				initialX, initialY, grid.X, grid.Y, turn+1)
			break
		}
	}

	assert.True(t, moved, "SpawnEnemyの敵は50ターン以内に移動するべき")
}

// TestFullTurnCycleWithAI はPlayer→AI→End→Playerのフルサイクルで敵が移動することを検証する
func TestFullTurnCycleWithAI(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Updaters = make(map[string]w.Updater)

	// 苔亀のwander移動は80%待機の確率的挙動のため、シードを固定してflaky化を防ぐ
	world.Config.RNG = rand.New(rand.NewPCG(1, 0))

	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	enemy, err := lifecycle.SpawnEnemy(world, 20, 20, "苔亀")
	require.NoError(t, err)

	// Waiting期間をスキップ
	solo := world.Components.SoloAI.Get(enemy)
	solo.SubState = gc.AIStateDriving
	solo.DurationSubStateTurns = 100

	initialGrid := world.Components.GridElement.Get(enemy)
	initialX, initialY := int(initialGrid.X), int(initialGrid.Y)

	sys := &TurnSystem{}

	moved := false
	for cycle := range 50 {
		turnState := query.GetTurnState(world)

		// PlayerTurn: APをマイナスにして自動でAIターンへ遷移させる
		turnState.Phase = gc.TurnPhasePlayer
		playerTB := world.Components.TurnBased.Get(player)
		playerTB.AP.Current = -1

		// PlayerTurn → AITurnへの自動遷移
		err = sys.Update(world)
		require.NoError(t, err)
		require.Equal(t, gc.TurnPhaseAI, turnState.Phase, "cycle %d: AITurnへ遷移するべき", cycle)

		// AITurn処理
		err = sys.Update(world)
		require.NoError(t, err)
		require.Equal(t, gc.TurnPhaseEnd, turnState.Phase, "cycle %d: TurnEndへ遷移するべき", cycle)

		// TurnEnd処理（AP回復）
		err = sys.Update(world)
		require.NoError(t, err)
		require.Equal(t, gc.TurnPhasePlayer, turnState.Phase, "cycle %d: PlayerTurnへ遷移するべき", cycle)

		grid := world.Components.GridElement.Get(enemy)
		if int(grid.X) != initialX || int(grid.Y) != initialY {
			moved = true
			t.Logf("フルサイクルで敵が移動: (%d,%d) → (%d,%d) at cycle %d",
				initialX, initialY, grid.X, grid.Y, cycle+1)
			break
		}
	}

	assert.True(t, moved, "フルターンサイクルで敵は50サイクル以内に移動するべき")
}

// TestPatrolMovement はPatrol移動パターンが直進と反転を正しく行うことを検証する
func TestPatrolMovement(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Updaters = make(map[string]w.Updater)

	_, err := lifecycle.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	// Patrol移動のAIエンティティを作成する。PatrolDirX=1で右に進む
	enemyX, enemyY := 20, 20
	enemy := world.ECS.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "パトロール敵"})
	world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemyData{})
	world.Components.GridElement.Add(enemy, &gc.GridElement{X: consts.Tile(enemyX), Y: consts.Tile(enemyY)})
	world.Components.SoloAI.Add(enemy, &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		Movement:              gc.SoloPatrol,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		OriginX:               enemyX,
		OriginY:               enemyY,
		PatrolDirX:            1,
		PatrolDirY:            0,
		ViewDistance:          5,
	})
	world.Components.TurnBased.Add(enemy, &gc.TurnBased{
		AP:    gc.IntPool{Current: 200, Max: 200},
		Speed: 100,
	})

	// 複数ターン実行して移動を確認する
	moved := false
	for turn := range 10 {
		tb := world.Components.TurnBased.Get(enemy)
		tb.AP.Current = 200

		turnState := query.GetTurnState(world)
		turnState.Phase = gc.TurnPhaseAI
		turnState.TurnNumber = turn + 1

		err := processAITurn(world)
		require.NoError(t, err)

		grid := world.Components.GridElement.Get(enemy)
		if int(grid.X) != enemyX || int(grid.Y) != enemyY {
			moved = true
			// Patrol移動なのでX座標が変化するはず
			t.Logf("Patrolエンティティが移動: (%d,%d) → (%d,%d)", enemyX, enemyY, grid.X, grid.Y)
			break
		}
	}

	assert.True(t, moved, "Patrolエンティティは移動するべき")
}

// TestTerritorialMovement はTerritorial移動パターンがスポーン地点から離れすぎないことを検証する
func TestTerritorialMovement(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Updaters = make(map[string]w.Updater)

	_, err := lifecycle.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	// Territorial移動のAIエンティティを作成する
	spawnX, spawnY := 20, 20
	enemy := world.ECS.NewEntity()
	world.Components.Name.Add(enemy, &gc.Name{Name: "縄張り敵"})
	world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemyData{})
	world.Components.GridElement.Add(enemy, &gc.GridElement{X: consts.Tile(spawnX), Y: consts.Tile(spawnY)})
	world.Components.SoloAI.Add(enemy, &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		Movement:              gc.SoloTerritorial,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		OriginX:               spawnX,
		OriginY:               spawnY,
		ViewDistance:          5,
	})
	world.Components.TurnBased.Add(enemy, &gc.TurnBased{
		AP:    gc.IntPool{Current: 200, Max: 200},
		Speed: 100,
	})

	// 多数のターンを実行して範囲内に留まることを検証する
	territorialRadius := 5
	for turn := range 100 {
		tb := world.Components.TurnBased.Get(enemy)
		tb.AP.Current = 200

		turnState := query.GetTurnState(world)
		turnState.Phase = gc.TurnPhaseAI
		turnState.TurnNumber = turn + 1

		err := processAITurn(world)
		require.NoError(t, err)

		grid := world.Components.GridElement.Get(enemy)
		dx := int(grid.X) - spawnX
		dy := int(grid.Y) - spawnY
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}

		assert.LessOrEqual(t, dx, territorialRadius,
			"turn %d: X座標がスポーン地点から%dタイル以内であるべき (pos=%d, spawn=%d)", turn, territorialRadius, grid.X, spawnX)
		assert.LessOrEqual(t, dy, territorialRadius,
			"turn %d: Y座標がスポーン地点から%dタイル以内であるべき (pos=%d, spawn=%d)", turn, territorialRadius, grid.Y, spawnY)
	}
}
