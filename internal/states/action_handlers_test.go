package states

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
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
		require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionUp))

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

				require.NoError(t, activity.ExecuteMoveAction(world, tt.direction))

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
			AP: gc.IntPool{Current: 50, Max: 50},
		})

		// 移動を実行（APがマイナスになる）
		err := activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// プレイヤーは移動している
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 9, int(gridAfter.Y))

		// APはマイナスになる
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		assert.Negative(t, turnBased.AP.Current, "移動コストでAPがマイナスになる")
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
		require.NoError(t, activity.ExecuteWaitAction(world))

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

func TestExecuteMoveActionWithEnemy(t *testing.T) {
	t.Parallel()

	t.Run("敵がいる位置への移動は攻撃になる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemyHP := world.Components.HP.Get(enemy).(*gc.HP)
		initialEnemyHP := enemyHP.Current

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
		assert.Less(t, enemyHP.Current, initialEnemyHP)
	})

	t.Run("冷えた状態でも敵への攻撃が可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(player).(*gc.HealthStatus)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		enemy, err := lifecycle.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		// APが0以上なら行動可能であることを確認
		tb := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		assert.GreaterOrEqual(t, tb.AP.Current, 0, "冷えた状態でもAPが0以上なら行動可能")
		enemyHP := world.Components.HP.Get(enemy).(*gc.HP)
		initialEnemyHP := enemyHP.Current

		// 攻撃を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorAttack, result.BehaviorName)
		assert.Less(t, enemyHP.Current, initialEnemyHP)
	})

	t.Run("冷えた状態で攻撃するとAPが消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(player).(*gc.HealthStatus)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		enemy, err := lifecycle.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		initialAP := turnBased.AP.Current
		enemyHP := world.Components.HP.Get(enemy).(*gc.HP)
		initialEnemyHP := enemyHP.Current

		// 攻撃を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorAttack, result.BehaviorName)
		assert.True(t, result.Success)
		assert.Less(t, turnBased.AP.Current, initialAP)
		assert.Less(t, enemyHP.Current, initialEnemyHP)
	})
}

func TestDeadEnemyInteraction(t *testing.T) {
	t.Parallel()

	t.Run("死亡した敵への移動は攻撃にならない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, 10, 9, "火の玉")
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemyHP := world.Components.HP.Get(enemy).(*gc.HP)
		enemyHP.Current = 1

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

func TestGetInteractionActions_Prop(t *testing.T) {
	t.Parallel()

	t.Run("攻撃可能なPropはメニューに表示される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "木箱"})
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.HP, &gc.HP{Max: 30, Current: 30})
		prop.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.MeleeInteraction{}},
		})

		actions := GetInteractionActions(world)
		require.Len(t, actions, 1)
		assert.Equal(t, "攻撃する(木箱)", actions[0].Label)
	})

	t.Run("敵対NPCもメニューに表示される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		enemy.AddComponent(world.Components.Name, &gc.Name{Name: "ゴブリン"})
		enemy.AddComponent(world.Components.SoloAI, &gc.SoloAI{
			CombatDefault: gc.CombatAttack,
			CombatCurrent: gc.CombatAttack,
			Movement:      gc.SoloRandom,
		})
		enemy.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.MeleeInteraction{}},
		})

		actions := GetInteractionActions(world)
		require.Len(t, actions, 1)
		assert.Equal(t, "攻撃する(ゴブリン)", actions[0].Label)
	})

	t.Run("方向キーでPropを自動攻撃しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 9})
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "木箱"})
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.HP, &gc.HP{Max: 30, Current: 30})
		prop.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
		prop.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.MeleeInteraction{}},
		})

		// 上に移動しようとする
		err := activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// Propに自動攻撃せず、移動もブロックされる
		grid := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 10, int(grid.X))
		assert.Equal(t, 10, int(grid.Y))
		hp := world.Components.HP.Get(prop).(*gc.HP)
		assert.Equal(t, 30, hp.Current, "Propに自動攻撃しないのでHPは減らない")
	})
}

func TestGetSameTileManualActions(t *testing.T) {
	t.Parallel()

	t.Run("同タイルのManualインタラクションを取得する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// SameTile+Manualのアイテムを配置
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		item.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.ItemInteraction{}},
		})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "テストアイテム"})

		actions := GetSameTileManualActions(world)
		assert.Len(t, actions, 1)
		assert.Contains(t, actions[0].Label, "テストアイテム")
	})

	t.Run("複数のManualインタラクションを全て取得する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// アイテム
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		item.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.ItemInteraction{}},
		})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

		// ポータル
		portal := world.Manager.NewEntity()
		portal.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		portal.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.PortalInteraction{PortalType: gc.PortalTypeNext}},
		})

		actions := GetSameTileManualActions(world)
		assert.Len(t, actions, 2, "アイテムとポータルの2つが取得される")
	})

	t.Run("別タイルのインタラクションは含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 隣接タイルのアイテム
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		item.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.ItemInteraction{}},
		})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "遠いアイテム"})

		actions := GetSameTileManualActions(world)
		assert.Empty(t, actions)
	})

	t.Run("OnCollisionインタラクションは含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// OnCollisionの扉（SameTileではなくAdjacentだが念のため）
		door := world.Manager.NewEntity()
		door.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		door.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.DoorInteraction{}},
		})
		door.AddComponent(world.Components.Door, &gc.Door{})

		actions := GetSameTileManualActions(world)
		assert.Empty(t, actions, "OnCollisionのインタラクションは含まれない")
	})

	t.Run("アイテムが2個以上あるとすべて拾うが先頭に追加される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		item1 := world.Manager.NewEntity()
		item1.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		item1.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.ItemInteraction{}},
		})
		item1.AddComponent(world.Components.Name, &gc.Name{Name: "木刀"})

		item2 := world.Manager.NewEntity()
		item2.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		item2.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.ItemInteraction{}},
		})
		item2.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

		actions := GetSameTileManualActions(world)
		require.Len(t, actions, 3, "すべて拾う + 個別2つ")
		assert.Equal(t, "すべて拾う", actions[0].Label)
		_, ok := actions[0].Interaction.(gc.ItemAllInteraction)
		assert.True(t, ok)
	})

	t.Run("アイテムが1個の場合はすべて拾うが追加されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		item.AddComponent(world.Components.Interactable, &gc.Interactable{
			Interactions: []gc.InteractionData{gc.ItemInteraction{}},
		})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "木刀"})

		actions := GetSameTileManualActions(world)
		require.Len(t, actions, 1)
		assert.Contains(t, actions[0].Label, "木刀")
	})

	t.Run("プレイヤーが存在しない場合はnil", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actions := GetSameTileManualActions(world)
		assert.Nil(t, actions)
	})
}
