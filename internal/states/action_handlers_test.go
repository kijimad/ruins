package states

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteMoveAction(t *testing.T) {
	t.Parallel()

	t.Run("正常な移動", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		// 移動を実行
		require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionUp))

		// 検証
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMove, result.BehaviorName)
		assert.True(t, result.Success)
		gridAfter := world.Components.GridElement.Get(player)
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
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

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

				player := world.ECS.NewEntity()
				world.Components.Player.Add(player, &gc.Player{})
				world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
				world.Components.TurnBased.Add(player, &gc.TurnBased{})

				require.NoError(t, activity.ExecuteMoveAction(world, tt.direction))

				gridAfter := world.Components.GridElement.Get(player)
				assert.Equal(t, tt.expectedX, int(gridAfter.X), "X座標が正しく移動するべき")
				assert.Equal(t, tt.expectedY, int(gridAfter.Y), "Y座標が正しく移動するべき")
			})
		}
	})

	t.Run("APがマイナスになっても移動は実行される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成（AP.Current >= 0 なら行動可能）
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.TurnBased.Add(player, &gc.TurnBased{
			AP: gc.IntPool{Current: 50, Max: 50},
		})

		// 移動を実行（APがマイナスになる）
		err := activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// プレイヤーは移動している
		gridAfter := world.Components.GridElement.Get(player)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 9, int(gridAfter.Y))

		// APはマイナスになる
		turnBased := world.Components.TurnBased.Get(player)
		assert.Negative(t, turnBased.AP.Current, "移動コストでAPがマイナスになる")
	})
}

func TestExecuteWaitAction(t *testing.T) {
	t.Parallel()

	t.Run("待機アクションの実行", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

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
		world.Resources.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		enemyHP := world.Components.HP.Get(enemy)
		initialEnemyHP := enemyHP.Current

		// 移動（攻撃）を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)
		assert.True(t, result.Success)
		gridAfter := world.Components.GridElement.Get(player)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 10, int(gridAfter.Y))
		assert.Less(t, enemyHP.Current, initialEnemyHP)
	})

	t.Run("冷えた状態でも敵への攻撃が可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(player)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		// APが0以上なら行動可能であることを確認
		tb := world.Components.TurnBased.Get(player)
		assert.GreaterOrEqual(t, tb.AP.Current, 0, "冷えた状態でもAPが0以上なら行動可能")
		enemyHP := world.Components.HP.Get(enemy)
		initialEnemyHP := enemyHP.Current

		// 攻撃を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)
		assert.Less(t, enemyHP.Current, initialEnemyHP)
	})

	t.Run("冷えた状態で攻撃するとAPが消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(player)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionHypothermia,
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		turnBased := world.Components.TurnBased.Get(player)
		initialAP := turnBased.AP.Current
		enemyHP := world.Components.HP.Get(enemy)
		initialEnemyHP := enemyHP.Current

		// 攻撃を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)
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
		world.Resources.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		world.Components.Dead.Add(enemy, &gc.Dead{})

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
		world.Resources.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		enemyHP := world.Components.HP.Get(enemy)
		enemyHP.Current = 1

		// 1回目: 攻撃で敵を倒す
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		assert.True(t, world.Components.Dead.Has(enemy))
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)

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

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		prop := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.Name.Add(prop, &gc.Name{Name: "木箱"})
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 30})
		world.Components.Interactable.Add(prop, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionMelee},
		})

		actions := GetInteractionActions(world)
		require.Len(t, actions, 1)
		assert.Equal(t, "Attack (木箱)", actions[0].Label)
	})

	t.Run("上り階段は前階へ転移するアクションを出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// 上り階段はプレイヤーと同じタイル。SameTile 発動
		stairs := world.ECS.NewEntity()
		world.Components.GridElement.Add(stairs, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(stairs, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionPortalPrev},
		})

		actions := GetInteractionActions(world)
		require.Len(t, actions, 1)
		assert.Equal(t, "Warp (previous floor)", actions[0].Label)
	})

	t.Run("敵対NPCもメニューに表示される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		enemy := world.ECS.NewEntity()
		world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.Name.Add(enemy, &gc.Name{Name: "ゴブリン"})
		world.Components.SoloAI.Add(enemy, &gc.SoloAI{
			CombatDefault: gc.CombatAttack,
			CombatCurrent: gc.CombatAttack,
			Movement:      gc.SoloRandom,
		})
		world.Components.Interactable.Add(enemy, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionMelee},
		})

		actions := GetInteractionActions(world)
		require.Len(t, actions, 1)
		assert.Equal(t, "Attack (ゴブリン)", actions[0].Label)
	})

	t.Run("同一スタックのアイテムは1行に束ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// 同一品種3個を同じタイルへ。1個1エンティティだが拾得行は1つに束ねる
		for range 3 {
			item := world.ECS.NewEntity()
			world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
			world.Components.LocationOnField.Add(item, &gc.LocationOnField{})
			world.Components.Interactable.Add(item, &gc.Interactable{
				Interactions: []gc.InteractionKind{gc.InteractionItem},
			})
			world.Components.RawID.Add(item, &gc.RawID{ID: "bread"})
			world.Components.Name.Add(item, &gc.Name{Name: "パン"})
		}

		actions := GetInteractionActions(world)
		require.Len(t, actions, 1, "同一スタックは1行。すべて拾うは1スタックなので出ない")
		assert.Contains(t, actions[0].Label, "3", "個数3がラベルに出る")
		assert.Equal(t, gc.InteractionItem, actions[0].Interaction)
	})

	t.Run("拾得と他種を併せ持つ実体は束ね行と個別行の両方に出る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// 拾えて調べられる合成実体。束ねるかは種別の宣言 StackBundled が決めるので、
		// 拾得行は束ね経路に、収納行は個別経路に、それぞれ1本ずつ出る
		hybrid := world.ECS.NewEntity()
		world.Components.GridElement.Add(hybrid, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 11}})
		world.Components.LocationOnField.Add(hybrid, &gc.LocationOnField{})
		world.Components.Interactable.Add(hybrid, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem, gc.InteractionStorage},
		})
		world.Components.RawID.Add(hybrid, &gc.RawID{ID: "mystery_box"})
		world.Components.Name.Add(hybrid, &gc.Name{Name: "ふしぎな箱"})

		actions := GetInteractionActions(world)
		require.Len(t, actions, 2, "収納の個別行と拾得の束ね行")
		kinds := []gc.InteractionKind{actions[0].Interaction, actions[1].Interaction}
		assert.Contains(t, kinds, gc.InteractionItem, "拾得行が消えない")
		assert.Contains(t, kinds, gc.InteractionStorage, "収納行も出る")
	})

	t.Run("方向キーでPropを自動攻撃しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})

		prop := world.ECS.NewEntity()
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 9}})
		world.Components.Name.Add(prop, &gc.Name{Name: "木箱"})
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 30})
		world.Components.BlockPass.Add(prop, &gc.BlockPass{})
		world.Components.Interactable.Add(prop, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionMelee},
		})

		// 上に移動しようとする
		err := activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 固定物に自動攻撃せず、移動もブロックされる
		grid := world.Components.GridElement.Get(player)
		assert.Equal(t, 10, int(grid.X))
		assert.Equal(t, 10, int(grid.Y))
		hp := world.Components.HP.Get(prop)
		assert.Equal(t, 30, hp.Current, "Propに自動攻撃しないのでHPは減らない")
	})
}

func TestGetSameTileManualActions(t *testing.T) {
	t.Parallel()

	t.Run("同タイルのManualインタラクションを取得する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// SameTile+Manualのアイテムを配置
		item := world.ECS.NewEntity()
		world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(item, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})
		world.Components.Name.Add(item, &gc.Name{Name: "テストアイテム"})

		actions := GetSameTileManualActions(world)
		assert.Len(t, actions, 1)
		assert.Contains(t, actions[0].Label, "テストアイテム")
	})

	t.Run("複数のManualインタラクションを全て取得する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// アイテム
		item := world.ECS.NewEntity()
		world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(item, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})
		world.Components.Name.Add(item, &gc.Name{Name: "回復薬"})

		// ポータル
		portal := world.ECS.NewEntity()
		world.Components.GridElement.Add(portal, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(portal, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionPortalNext},
		})

		actions := GetSameTileManualActions(world)
		assert.Len(t, actions, 2, "アイテムとポータルの2つが取得される")
	})

	t.Run("別タイルのインタラクションは含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// 隣接タイルのアイテム
		item := world.ECS.NewEntity()
		world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.Interactable.Add(item, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})
		world.Components.Name.Add(item, &gc.Name{Name: "遠いアイテム"})

		actions := GetSameTileManualActions(world)
		assert.Empty(t, actions)
	})

	t.Run("OnCollisionインタラクションは含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// OnCollisionの扉（SameTileではなくAdjacentだが念のため）
		door := world.ECS.NewEntity()
		world.Components.GridElement.Add(door, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(door, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionDoor},
		})
		world.Components.Door.Add(door, &gc.Door{})

		actions := GetSameTileManualActions(world)
		assert.Empty(t, actions, "OnCollisionのインタラクションは含まれない")
	})

	t.Run("アイテムが2個以上あるとすべて拾うが先頭に追加される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// 別品種は別スタックなので個別行が2つ出る。RawID が同定キーになる
		item1 := world.ECS.NewEntity()
		world.Components.GridElement.Add(item1, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(item1, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})
		world.Components.RawID.Add(item1, &gc.RawID{ID: "wooden_sword"})
		world.Components.Name.Add(item1, &gc.Name{Name: "木刀"})

		item2 := world.ECS.NewEntity()
		world.Components.GridElement.Add(item2, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(item2, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})
		world.Components.RawID.Add(item2, &gc.RawID{ID: "healing_potion"})
		world.Components.Name.Add(item2, &gc.Name{Name: "回復薬"})

		actions := GetSameTileManualActions(world)
		require.Len(t, actions, 3, "すべて拾う + 個別2つ")
		assert.Equal(t, "Pick up all", actions[0].Label)
		ok := actions[0].Interaction == gc.InteractionItemAll
		assert.True(t, ok)
	})

	t.Run("同一スタックのアイテムは1行に束ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// 同一品種を2個、同じタイルへ。1個1エンティティだが1スタックに束ねる
		for range 2 {
			item := world.ECS.NewEntity()
			world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
			world.Components.LocationOnField.Add(item, &gc.LocationOnField{})
			world.Components.Interactable.Add(item, &gc.Interactable{
				Interactions: []gc.InteractionKind{gc.InteractionItem},
			})
			world.Components.RawID.Add(item, &gc.RawID{ID: "biscuit"})
			world.Components.Name.Add(item, &gc.Name{Name: "ビスケット"})
		}

		actions := GetSameTileManualActions(world)
		require.Len(t, actions, 1, "同一スタックは1行だけ。すべて拾うは1スタックなので出ない")
		assert.Contains(t, actions[0].Label, "2", "個数2がラベルに出る")
		assert.Equal(t, gc.InteractionItem, actions[0].Interaction)
	})

	t.Run("アイテムが1個の場合はすべて拾うが追加されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		item := world.ECS.NewEntity()
		world.Components.GridElement.Add(item, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(item, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})
		world.Components.Name.Add(item, &gc.Name{Name: "木刀"})

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

// TestGetInteractionActions_着火 は、火種の所持と隣接の燃焼物を条件に着火アクションが
// タイル単位で1つ出ること、足元や遠いタイルは対象外になることを確認する。
func TestGetInteractionActions_着火(t *testing.T) {
	t.Parallel()

	// 火種の有無だけを見るため、初期装備の松明が付く SpawnPlayer は使わず素のプレイヤーを組む。
	// 松明は火種を兼ねるので標準装備では常に着火できてしまい、火種なしの検証ができない
	setup := func(t *testing.T) (w.World, ecs.Entity) {
		t.Helper()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
		return world, player
	}
	addFieldFuel := func(world w.World, x, y consts.Tile, name string) {
		e := world.ECS.NewEntity()
		world.Components.Name.Add(e, &gc.Name{Name: name})
		// 熱量は材質×重量から導く。WOOD 200/kg × 50g = 10
		world.Components.Material.Add(e, &gc.Material{Kind: oapi.WOOD})
		world.Components.Weight.Add(e, &gc.Weight{Milligram: 50 * consts.MilligramPerGram})
		world.Components.GridElement.Add(e, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		world.Components.LocationOnField.Add(e, &gc.LocationOnField{})
	}
	giveFireStarter := func(world w.World, player ecs.Entity) {
		s := world.ECS.NewEntity()
		world.Components.FireStarter.Add(s, &gc.FireStarter{})
		world.Components.LocationInBackpack.Add(s, &gc.LocationInBackpack{Owner: player})
	}
	countIgnite := func(actions []InteractionAction) int {
		n := 0
		for _, a := range actions {
			if a.Interaction == gc.InteractionIgnite {
				n++
			}
		}
		return n
	}

	t.Run("火種が無ければ着火は出ない", func(t *testing.T) {
		t.Parallel()
		world, _ := setup(t)
		addFieldFuel(world, 6, 5, "wood")
		assert.Equal(t, 0, countIgnite(GetInteractionActions(world)))
	})

	t.Run("火種を持ち隣接に燃料があれば着火が1つ出る", func(t *testing.T) {
		t.Parallel()
		world, player := setup(t)
		giveFireStarter(world, player)
		addFieldFuel(world, 6, 5, "wood")
		assert.Equal(t, 1, countIgnite(GetInteractionActions(world)))
	})

	t.Run("同じタイルに複数の燃料があっても着火は1つに束ねる", func(t *testing.T) {
		t.Parallel()
		world, player := setup(t)
		giveFireStarter(world, player)
		addFieldFuel(world, 6, 5, "a_wood")
		addFieldFuel(world, 6, 5, "b_wood")
		assert.Equal(t, 1, countIgnite(GetInteractionActions(world)))
	})

	t.Run("足元の燃料は着火の対象にしない", func(t *testing.T) {
		t.Parallel()
		world, player := setup(t)
		giveFireStarter(world, player)
		addFieldFuel(world, 5, 5, "wood")
		assert.Equal(t, 0, countIgnite(GetInteractionActions(world)))
	})

	t.Run("2タイル離れた燃料は隣接でないので着火は出ない", func(t *testing.T) {
		t.Parallel()
		world, player := setup(t)
		giveFireStarter(world, player)
		addFieldFuel(world, 7, 5, "wood")
		assert.Equal(t, 0, countIgnite(GetInteractionActions(world)))
	})
}

// TestGetInteractionActions_給油 は、バックパックに燃料を持ち隣接に燃えている火があるとき
// 給油アクションが出ること、燃料なし・非隣接・非燃焼では出ないことを確認する。
func TestGetInteractionActions_給油(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (w.World, ecs.Entity) {
		t.Helper()
		world := testutil.InitTestWorld(t)
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
		return world, player
	}
	addBurningFire := func(world w.World, x, y consts.Tile) {
		fire := world.ECS.NewEntity()
		world.Components.GridElement.Add(fire, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}})
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 5})
	}
	giveFuel := func(world w.World, player ecs.Entity) {
		f := world.ECS.NewEntity()
		// 熱量は材質×重量から導く。WOOD 200/kg × 50g = 10
		world.Components.Material.Add(f, &gc.Material{Kind: oapi.WOOD})
		world.Components.Weight.Add(f, &gc.Weight{Milligram: 50 * consts.MilligramPerGram})
		world.Components.LocationInBackpack.Add(f, &gc.LocationInBackpack{Owner: player})
	}
	countFeed := func(actions []InteractionAction) int {
		n := 0
		for _, a := range actions {
			if a.Interaction == gc.InteractionFeedFuel {
				n++
			}
		}
		return n
	}

	t.Run("燃料が無くても隣接の火なら給油メニューを開ける", func(t *testing.T) {
		t.Parallel()
		world, _ := setup(t)
		addBurningFire(world, 6, 5)
		// 火の残ターン数を確認できるよう、燃料の有無に関わらずメニューは開ける
		assert.Equal(t, 1, countFeed(GetInteractionActions(world)))
	})

	t.Run("燃料を持ち隣接に火があれば給油が出る", func(t *testing.T) {
		t.Parallel()
		world, player := setup(t)
		giveFuel(world, player)
		addBurningFire(world, 6, 5)
		assert.Equal(t, 1, countFeed(GetInteractionActions(world)))
	})

	t.Run("2タイル離れた火には給油しない", func(t *testing.T) {
		t.Parallel()
		world, player := setup(t)
		giveFuel(world, player)
		addBurningFire(world, 7, 5)
		assert.Equal(t, 0, countFeed(GetInteractionActions(world)))
	})

	t.Run("燃えていない火には給油しない", func(t *testing.T) {
		t.Parallel()
		world, player := setup(t)
		giveFuel(world, player)
		// Burning を持たない冷えた火エンティティ
		cold := world.ECS.NewEntity()
		world.Components.GridElement.Add(cold, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 6, Y: 5}})
		assert.Equal(t, 0, countFeed(GetInteractionActions(world)))
	})
}
