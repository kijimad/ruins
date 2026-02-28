package states

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/actions"
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
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// マップサイズを設定（移動判定に必要）
		world.Resources.Dungeon.Level.TileWidth = 50
		world.Resources.Dungeon.Level.TileHeight = 50

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 移動前の座標を確認
		gridBefore := world.Components.GridElement.Get(player).(*gc.GridElement)
		initialX := int(gridBefore.X)
		initialY := int(gridBefore.Y)

		// 北に移動
		assert.NoError(t, ExecuteMoveAction(world, gc.DirectionUp))

		// 移動後の座標を確認
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, initialX, int(gridAfter.X), "X座標は変化しないべき")
		assert.Equal(t, initialY-1, int(gridAfter.Y), "Y座標が1減るべき")
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// プレイヤーなしで移動を試みる（エラーが返ることを確認）
		assert.Error(t, ExecuteMoveAction(world, gc.DirectionUp))
	})

	t.Run("GridElementがない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// 移動を試みる（パニックしないことを確認）
		assert.NoError(t, ExecuteMoveAction(world, gc.DirectionUp))
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
				world.Resources.ActivityManager = actions.NewActivityManager(nil)

				// マップサイズを設定（移動判定に必要）
				world.Resources.Dungeon.Level.TileWidth = 50
				world.Resources.Dungeon.Level.TileHeight = 50

				player := world.Manager.NewEntity()
				player.AddComponent(world.Components.Player, &gc.Player{})
				player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
				player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

				assert.NoError(t, ExecuteMoveAction(world, tt.direction))

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
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

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
		err := ExecuteMoveAction(world, gc.DirectionUp)
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
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 待機アクションを実行
		assert.NoError(t, ExecuteWaitAction(world))

		assert.True(t, player.HasComponent(world.Components.Player), "プレイヤーが存在するべき")
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// プレイヤーなしで待機を試みる（エラーが返ることを確認）
		assert.Error(t, ExecuteWaitAction(world))
	})
}

func TestExecuteEnterAction(t *testing.T) {
	t.Parallel()

	t.Run("アイテムがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

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
		assert.NoError(t, ExecuteEnterAction(world))

		// Enterアクションが実行されることを確認（パニックしない）
		assert.True(t, player.HasComponent(world.Components.Player), "プレイヤーが存在するべき")
	})

	t.Run("ワープホールがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// Enterアクションを実行（処理が呼ばれることを期待）
		assert.NoError(t, ExecuteEnterAction(world))

		// パニックしないことを確認
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// プレイヤーなしでEnterを試みる（エラーが返ることを確認）
		assert.Error(t, ExecuteEnterAction(world))
	})

	t.Run("GridElementがない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// Enterを試みる（パニックしないことを確認）
		assert.NoError(t, ExecuteEnterAction(world))
		// エラーにならず何も起きないべき
	})

	t.Run("何もない場所でEnter", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// Enterアクションを実行
		assert.NoError(t, ExecuteEnterAction(world))

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
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// 固定シードで決定的な攻撃結果にする
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		// プレイヤーを作成
		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// 北隣に敵を作成
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)

		initialPlayerX := int(world.Components.GridElement.Get(player).(*gc.GridElement).X)
		initialPlayerY := int(world.Components.GridElement.Get(player).(*gc.GridElement).Y)
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 北に移動（敵がいる方向）
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// プレイヤーが移動していないことを確認（攻撃したため）
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, initialPlayerX, int(gridAfter.X), "攻撃時はX座標が変化しないべき")
		assert.Equal(t, initialPlayerY, int(gridAfter.Y), "攻撃時はY座標が変化しないべき")

		// 敵のHPが減少していることを確認
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP, "攻撃後は敵のHPが減少しているべき")
	})

	t.Run("冷えた状態でも敵への攻撃が可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)
		turnManager := world.Resources.TurnManager.(*turns.TurnManager)

		// 固定シードで攻撃が必ず命中するようにする
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		// プレイヤーを作成
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

		// 北隣に敵を作成
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)

		// CanEntityActがtrueを返すことを確認
		canAct := turnManager.CanEntityAct(world, player, 100)
		assert.True(t, canAct, "冷えた状態でもAPが0以上なら行動可能")

		initialPlayerX := int(world.Components.GridElement.Get(player).(*gc.GridElement).X)
		initialPlayerY := int(world.Components.GridElement.Get(player).(*gc.GridElement).Y)
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 北に移動（敵がいる方向）- これが攻撃になるはず
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 攻撃したためプレイヤーが移動していないことを確認
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, initialPlayerX, int(gridAfter.X), "攻撃時はX座標が変化しないべき")
		assert.Equal(t, initialPlayerY, int(gridAfter.Y), "攻撃時はY座標が変化しないべき")

		// 攻撃によって敵のHPが減少していることを確認する
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP, "攻撃後は敵のHPが減少しているべき")
	})

	t.Run("冷えた状態で攻撃するとAPが消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Resources.ActivityManager = actions.NewActivityManager(nil)

		// 固定シードで決定的な動作にする
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		// プレイヤーを作成
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

		// 北隣に敵を作成
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)

		// 初期値を記録
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		initialAP := turnBased.AP.Current
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 北に移動（敵がいる方向）- これが攻撃になるはず
		err = ExecuteMoveAction(world, gc.DirectionUp)
		assert.NoError(t, err, "攻撃アクションはエラーなく実行されるべき")

		// APが消費されていることを確認
		assert.Less(t, turnBased.AP.Current, initialAP, "攻撃後はAPが減少しているべき")

		// 敵のHPが減少していることを確認
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP, "攻撃後は敵のHPが減少しているべき")
	})
}

func TestCheckTileEvents(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーエンティティの場合のみイベントチェック", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// checkTileEventsを呼び出し（パニックしないことを確認）
		checkTileEvents(world, player, 10, 10)
	})

	t.Run("非プレイヤーエンティティの場合はイベントチェックしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 敵を作成
		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// checkTileEventsを呼び出し（パニックしないことを確認）
		checkTileEvents(world, enemy, 10, 10)
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
		interactable, entity := getInteractableAtSameTile(world, playerGrid)

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
		interactable, _ := getInteractableAtSameTile(world, playerGrid)

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
		interactable, _ := getInteractableAtSameTile(world, playerGrid)

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
		interactable, entity := getInteractableInRange(world, playerGrid)

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
		interactable, _ := getInteractableInRange(world, playerGrid)

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

		// 履歴記録用スライスを設定
		var history []actions.ActivityHistoryEntry
		manager := actions.NewActivityManager(nil)
		manager.History = &history
		world.Resources.ActivityManager = manager

		// マップサイズを設定して移動可能にする
		world.Resources.Dungeon.Level.TileWidth = 50
		world.Resources.Dungeon.Level.TileHeight = 50

		// 固定シードで決定的な動作にする
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		// プレイヤーを作成
		_, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// 北隣に敵を作成して即座に死亡させる
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemy.AddComponent(world.Components.Dead, &gc.Dead{})

		// 北に移動（死亡した敵がいる方向）
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 履歴から実行されたアクティビティを検証
		require.Len(t, history, 1, "1つのアクティビティが実行されるべき")
		assert.Equal(t, "Move", history[0].ActivityName, "Moveが実行されるべき")
		assert.True(t, history[0].Success, "移動は成功するべき")
	})

	t.Run("敵を倒した後の再移動はMoveになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()

		// 履歴記録用スライスを設定
		var history []actions.ActivityHistoryEntry
		manager := actions.NewActivityManager(nil)
		manager.History = &history
		world.Resources.ActivityManager = manager

		// マップサイズを設定して移動可能にする
		world.Resources.Dungeon.Level.TileWidth = 50
		world.Resources.Dungeon.Level.TileHeight = 50

		// 固定シードで決定的な動作にする
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		// プレイヤーを作成
		_, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// 北隣に敵を作成
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)

		// 敵のHPを1にして一撃で倒せるようにする
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		enemyPools.HP.Current = 1

		// 1回目: 攻撃で敵を倒す
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		assert.True(t, enemy.HasComponent(world.Components.Dead), "敵は死亡しているべき")

		// 1回目は InteractionActivate（攻撃）
		require.Len(t, history, 1)
		assert.Equal(t, "InteractionActivate", history[0].ActivityName, "1回目は攻撃")

		// 2回目: 死亡した敵がいた場所への移動
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 2回目は Move（死亡した敵は無視される）
		require.Len(t, history, 2)
		assert.Equal(t, "Move", history[1].ActivityName, "2回目は移動")
		assert.True(t, history[1].Success, "移動は成功するべき")
	})
}
