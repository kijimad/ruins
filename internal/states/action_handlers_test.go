package states

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteMoveAction(t *testing.T) {
	t.Parallel()

	t.Run("正常な移動", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 移動を実行
		assert.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionUp))

		// 検証
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMove, result.BehaviorName)
		assert.True(t, result.Success)
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 9, int(gridAfter.Y))
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしで移動を試みる（エラーが返ることを確認）
		assert.Error(t, activity.ExecuteMoveAction(world, gc.DirectionUp))
	})

	t.Run("GridElementがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		assert.Error(t, activity.ExecuteMoveAction(world, gc.DirectionUp))
	})

	t.Run("8方向の移動", func(t *testing.T) {
		t.Parallel()

		directions := []struct {
			name      string
			direction gc.Direction
			expectedX int
			expectedY int
		}{
			{"北", gc.DirectionUp, 10, 9},
			{"南", gc.DirectionDown, 10, 11},
			{"東", gc.DirectionRight, 11, 10},
			{"西", gc.DirectionLeft, 9, 10},
			{"北東", gc.DirectionUpRight, 11, 9},
			{"北西", gc.DirectionUpLeft, 9, 9},
			{"南東", gc.DirectionDownRight, 11, 11},
			{"南西", gc.DirectionDownLeft, 9, 11},
		}

		for _, tt := range directions {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				world := testutil.InitTestWorld(t)

				player := world.Manager.NewEntity()
				player.AddComponent(world.Components.Player, &gc.Player{})
				player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
				player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

				assert.NoError(t, activity.ExecuteMoveAction(world, tt.direction))

				gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
				assert.Equal(t, tt.expectedX, int(gridAfter.X), "X座標が正しく移動するべき")
				assert.Equal(t, tt.expectedY, int(gridAfter.Y), "Y座標が正しく移動するべき")
			})
		}
	})

	t.Run("APがマイナスになっても移動は実行される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成（AP.Current >= 0 なら行動可能）
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
			AP: gc.Pool{Current: 50, Max: 50},
		})

		// 移動を実行（APがマイナスになる）
		err := activity.ExecuteMoveAction(world, gc.DirectionUp)
		assert.NoError(t, err)

		// プレイヤーは移動している
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 9, int(gridAfter.Y))

		// APはマイナスになる
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		assert.Less(t, turnBased.AP.Current, 0, "移動コストでAPがマイナスになる")
	})
}

func TestExecuteWaitAction(t *testing.T) {
	t.Parallel()

	t.Run("待機アクションの実行", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 待機を実行
		assert.NoError(t, activity.ExecuteWaitAction(world))

		// 検証
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorWait, result.BehaviorName)
		assert.True(t, result.Success)
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしで待機を試みる（エラーが返ることを確認）
		assert.Error(t, activity.ExecuteWaitAction(world))
	})
}

func TestExecuteEnterAction(t *testing.T) {
	t.Parallel()

	t.Run("アイテムがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 同じ位置にアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{})
		item.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		item.AddComponent(world.Components.ItemLocationOnField, &gc.ItemLocationOnField)
		item.AddComponent(world.Components.Name, &gc.Name{Name: "テストアイテム"})

		// Enterアクションを実行
		assert.NoError(t, activity.ExecuteEnterAction(world))

		// Enterアクションが実行されることを確認（パニックしない）
		assert.True(t, player.HasComponent(world.Components.Player), "プレイヤーが存在するべき")
	})

	t.Run("ワープホールがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// Enterアクションを実行（処理が呼ばれることを期待）
		assert.NoError(t, activity.ExecuteEnterAction(world))

		// パニックしないことを確認
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしでEnterを試みる（エラーが返ることを確認）
		assert.Error(t, activity.ExecuteEnterAction(world))
	})

	t.Run("GridElementがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		assert.Error(t, activity.ExecuteEnterAction(world))
	})

	t.Run("何もない場所でEnter", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// Enterアクションを実行
		assert.NoError(t, activity.ExecuteEnterAction(world))

		// 何も起きないことを確認（パニックしない）
		assert.True(t, player.HasComponent(world.Components.Player), "プレイヤーが存在するべき")
	})
}

func TestExecuteMoveActionWithEnemy(t *testing.T) {
	t.Parallel()

	t.Run("敵がいる位置への移動は攻撃になる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 移動（攻撃）を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorAttack, result.BehaviorName)
		assert.True(t, result.Success)
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 10, int(gridAfter.Y))
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP)
	})

	t.Run("冷えた状態でも敵への攻撃が可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(player).(*gc.HealthStatus)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			hs.Parts[i].SetCondition(gc.HealthCondition{
				Type:     gc.ConditionHypothermia,
				Severity: gc.SeveritySevere,
				Timer:    90,
			})
		}

		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		// APが0以上なら行動可能であることを確認
		tb := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		assert.GreaterOrEqual(t, tb.AP.Current, 0, "冷えた状態でもAPが0以上なら行動可能")
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 攻撃を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorAttack, result.BehaviorName)
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP)
	})

	t.Run("冷えた状態で攻撃するとAPが消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(player).(*gc.HealthStatus)
		for i := 0; i < int(gc.BodyPartCount); i++ {
			hs.Parts[i].SetCondition(gc.HealthCondition{
				Type:     gc.ConditionHypothermia,
				Severity: gc.SeveritySevere,
				Timer:    90,
			})
		}

		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		initialAP := turnBased.AP.Current
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 攻撃を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorAttack, result.BehaviorName)
		assert.True(t, result.Success)
		assert.Less(t, turnBased.AP.Current, initialAP)
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP)
	})
}

func TestDeadEnemyInteraction(t *testing.T) {
	t.Parallel()

	t.Run("死亡した敵への移動は攻撃にならない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemy.AddComponent(world.Components.Dead, &gc.Dead{})

		// 移動を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMove, result.BehaviorName)
		assert.True(t, result.Success)
	})

	t.Run("敵を倒した後の再移動はMoveになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		enemyPools.HP.Current = 1

		// 1回目: 攻撃で敵を倒す
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		assert.True(t, enemy.HasComponent(world.Components.Dead))
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorAttack, result.BehaviorName)

		// 2回目: 死亡した敵がいた場所への移動
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		result = activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMove, result.BehaviorName)
		assert.True(t, result.Success)
	})
}
