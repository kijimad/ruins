package states

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/turns"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteMoveAction(t *testing.T) {
	t.Parallel()

	t.Run("正常な移動", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := activity.NewManager(nil)
		world.Resources.ActivityManager = manager
		world.Resources.Dungeon.Level.TileWidth = 50
		world.Resources.Dungeon.Level.TileHeight = 50

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 移動を実行
		var history []activity.HistoryEntry
		manager.History = &history
		assert.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionUp))

		// 検証
		require.Len(t, history, 1)
		assert.Equal(t, "Move", history[0].Activity.String())
		assert.True(t, history[0].Success)
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 9, int(gridAfter.Y))
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

		// プレイヤーなしで移動を試みる（エラーが返ることを確認）
		assert.Error(t, activity.ExecuteMoveAction(world, gc.DirectionUp))
	})

	t.Run("GridElementがない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// 移動を試みる（パニックしないことを確認）
		assert.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionUp))
		// エラーにならず何も起きないべき
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
				world.Resources.TurnManager = turns.NewTurnManager()
				world.Resources.ActivityManager = activity.NewManager(nil)

				// マップサイズを設定（移動判定に必要）
				world.Resources.Dungeon.Level.TileWidth = 50
				world.Resources.Dungeon.Level.TileHeight = 50

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

	t.Run("最大APが不足している場合は移動できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

		// プレイヤーを作成（AP.Max < 100）
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
			AP: gc.Pool{Current: 50, Max: 50}, // 移動に必要なAP(100)より少ない
		})

		// 移動前の座標とAPを記録
		initialX := int(world.Components.GridElement.Get(player).(*gc.GridElement).X)
		initialY := int(world.Components.GridElement.Get(player).(*gc.GridElement).Y)
		initialAP := world.Components.TurnBased.Get(player).(*gc.TurnBased).AP.Current

		// 移動を試みる（AP.Maxが不足しているため、Validateでgamelogに出力され、移動は実行されない）
		err := activity.ExecuteMoveAction(world, gc.DirectionUp)
		assert.NoError(t, err, "AP.Max不足時もエラーは返さない（gamelog出力のみ）")

		// プレイヤーは移動していない（AP.Maxが不足している場合は移動しない）
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, initialX, int(gridAfter.X), "AP.Max不足時はX座標は変化しない")
		assert.Equal(t, initialY, int(gridAfter.Y), "AP.Max不足時はY座標は変化しない")

		// APも消費されない
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		assert.Equal(t, initialAP, turnBased.AP.Current, "AP.Max不足時はAPも消費されない")
	})
}

func TestExecuteWaitAction(t *testing.T) {
	t.Parallel()

	t.Run("待機アクションの実行", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := activity.NewManager(nil)
		world.Resources.ActivityManager = manager

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 待機を実行
		var history []activity.HistoryEntry
		manager.History = &history
		assert.NoError(t, activity.ExecuteWaitAction(world))

		// 検証
		require.Len(t, history, 1)
		assert.Equal(t, "Wait", history[0].Activity.String())
		assert.True(t, history[0].Success)
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

		// プレイヤーなしで待機を試みる（エラーが返ることを確認）
		assert.Error(t, activity.ExecuteWaitAction(world))
	})
}

func TestExecuteEnterAction(t *testing.T) {
	t.Parallel()

	t.Run("アイテムがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

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
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

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
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

		// プレイヤーなしでEnterを試みる（エラーが返ることを確認）
		assert.Error(t, activity.ExecuteEnterAction(world))
	})

	t.Run("GridElementがない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// Enterを試みる（パニックしないことを確認）
		assert.NoError(t, activity.ExecuteEnterAction(world))
		// エラーにならず何も起きないべき
	})

	t.Run("何もない場所でEnter", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = activity.NewManager(nil)

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
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))
		manager := activity.NewManager(nil)
		world.Resources.ActivityManager = manager

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 移動（攻撃）を実行
		var history []activity.HistoryEntry
		manager.History = &history
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		require.Len(t, history, 1)
		assert.Equal(t, "Attack", history[0].Activity.String())
		assert.True(t, history[0].Success)
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 10, int(gridAfter.Y))
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP)
	})

	t.Run("冷えた状態でも敵への攻撃が可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		turnManager := world.Resources.TurnManager.(*turns.TurnManager)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))
		manager := activity.NewManager(nil)
		world.Resources.ActivityManager = manager

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
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
		canAct := turnManager.CanEntityAct(world, player, 100)
		assert.True(t, canAct, "冷えた状態でもAPが0以上なら行動可能")
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 攻撃を実行
		var history []activity.HistoryEntry
		manager.History = &history
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		require.Len(t, history, 1)
		assert.Equal(t, "Attack", history[0].Activity.String())
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP)
	})

	t.Run("冷えた状態で攻撃するとAPが消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))
		manager := activity.NewManager(nil)
		world.Resources.ActivityManager = manager

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
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
		var history []activity.HistoryEntry
		manager.History = &history
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		require.Len(t, history, 1)
		assert.Equal(t, "Attack", history[0].Activity.String())
		assert.True(t, history[0].Success)
		assert.Less(t, turnBased.AP.Current, initialAP)
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP)
	})
}

func TestGetInteractableAtSameTile(t *testing.T) {
	t.Parallel()

	t.Run("同じタイルのInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Interactableエンティティを作成
		interactableEntity := world.Manager.NewEntity()
		interactableEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		interactableEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{},
		})

		playerGrid := &gc.GridElement{X: 5, Y: 5}
		interactable, entity := activity.GetInteractableAtSameTile(world, playerGrid)

		assert.NotNil(t, interactable)
		assert.Equal(t, interactableEntity, entity)
	})

	t.Run("死亡エンティティはInteractable対象から除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 死亡したInteractableエンティティを作成
		deadEntity := world.Manager.NewEntity()
		deadEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		deadEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{},
		})
		deadEntity.AddComponent(world.Components.Dead, &gc.Dead{})

		playerGrid := &gc.GridElement{X: 5, Y: 5}
		interactable, _ := activity.GetInteractableAtSameTile(world, playerGrid)

		// 死亡エンティティは見つからない
		assert.Nil(t, interactable)
	})

	t.Run("異なるタイルのInteractableは取得されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 別のタイルにInteractableエンティティを作成
		interactableEntity := world.Manager.NewEntity()
		interactableEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		interactableEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{},
		})

		playerGrid := &gc.GridElement{X: 5, Y: 5}
		interactable, _ := activity.GetInteractableAtSameTile(world, playerGrid)

		assert.Nil(t, interactable)
	})
}

func TestGetInteractableInRange(t *testing.T) {
	t.Parallel()

	t.Run("範囲内のInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 隣接タイルにInteractableエンティティを作成
		interactableEntity := world.Manager.NewEntity()
		interactableEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 6, Y: 5})
		interactableEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{}, // ActivationRangeはAdjacentを返す
		})

		playerGrid := &gc.GridElement{X: 5, Y: 5}
		interactable, entity := activity.GetInteractableInRange(world, playerGrid)

		assert.NotNil(t, interactable)
		assert.Equal(t, interactableEntity, entity)
	})

	t.Run("死亡エンティティは範囲内でも除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 隣接タイルに死亡したInteractableエンティティを作成
		deadEntity := world.Manager.NewEntity()
		deadEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 6, Y: 5})
		deadEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{},
		})
		deadEntity.AddComponent(world.Components.Dead, &gc.Dead{})

		playerGrid := &gc.GridElement{X: 5, Y: 5}
		interactable, _ := activity.GetInteractableInRange(world, playerGrid)

		// 死亡エンティティは見つからない
		assert.Nil(t, interactable)
	})
}

func TestDeadEnemyInteraction(t *testing.T) {
	t.Parallel()

	t.Run("死亡した敵への移動は攻撃にならない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.Dungeon.Level.TileWidth = 50
		world.Resources.Dungeon.Level.TileHeight = 50
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))
		manager := activity.NewManager(nil)
		world.Resources.ActivityManager = manager

		_, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemy.AddComponent(world.Components.Dead, &gc.Dead{})

		// 移動を実行
		var history []activity.HistoryEntry
		manager.History = &history
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証
		require.Len(t, history, 1)
		assert.Equal(t, "Move", history[0].Activity.String())
		assert.True(t, history[0].Success)
	})

	t.Run("敵を倒した後の再移動はMoveになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.Dungeon.Level.TileWidth = 50
		world.Resources.Dungeon.Level.TileHeight = 50
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))
		manager := activity.NewManager(nil)
		world.Resources.ActivityManager = manager

		_, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		enemyPools.HP.Current = 1

		// 攻撃→移動を実行
		var history []activity.HistoryEntry
		manager.History = &history

		// 1回目: 攻撃で敵を倒す
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		assert.True(t, enemy.HasComponent(world.Components.Dead))
		require.Len(t, history, 1)
		assert.Equal(t, "Attack", history[0].Activity.String())

		// 2回目: 死亡した敵がいた場所への移動
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		require.Len(t, history, 2)
		assert.Equal(t, "Move", history[1].Activity.String())
		assert.True(t, history[1].Success)
	})
}
