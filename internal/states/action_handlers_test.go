package states

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/rawschema"
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

	t.Run("冷えた状態でも攻撃が成立しAPを消費する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 重度の低体温を設定
		hs := world.Components.HealthStatus.Get(player)
		hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
			Type:     gc.ConditionType("hypothermia"),
			Severity: gc.SeveritySevere,
			Timer:    90,
		})

		// 上方向に攻撃対象を置く。ハンドルは使わないので捨てる
		_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		turnBased := world.Components.TurnBased.Get(player)
		initialAP := turnBased.AP.Current

		// 攻撃を実行
		err = activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: 冷えていても近接攻撃が成立し AP を消費する。命中は身体機能で下がるので当否は問わない
		result := activity.GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)
		assert.True(t, result.Success)
		assert.Less(t, turnBased.AP.Current, initialAP)
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

	t.Run("隣接の扉にだけ操作を出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, gc.DoorOrientationHorizontal)
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", 10, 11)
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "grass", 11, 10)
		require.NoError(t, err)
		_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 9, Y: 10}, "fireball")
		require.NoError(t, err)
		_, err = lifecycle.SpawnFieldItem(world, "bread", 10, 10, 1)
		require.NoError(t, err)

		assert.Equal(t, []InteractionAction{
			{Label: "Open (up)", Target: door, Interaction: gc.InteractionDoor},
		}, GetInteractionActions(world))
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

	t.Run("隣接では直上発動の拾得を出さず隣接発動の会話だけ出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		// 拾得(SameTile)と会話(Adjacent)を併せ持つ実体を隣接タイルへ。発動範囲は相互作用ごとに
		// 判定するので、隣接では隣接発動の会話だけが出て、直上でしか発動しない拾得は出ない
		hybrid := world.ECS.NewEntity()
		world.Components.GridElement.Add(hybrid, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 11}})
		world.Components.LocationOnField.Add(hybrid, &gc.LocationOnField{})
		world.Components.Interactable.Add(hybrid, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem, gc.InteractionTalk},
		})
		world.Components.RawID.Add(hybrid, &gc.RawID{ID: "mystery_box"})
		world.Components.Name.Add(hybrid, &gc.Name{Name: "ふしぎな箱"})

		actions := GetInteractionActions(world)
		assert.Equal(t, []InteractionAction{
			{Label: "Talk (ふしぎな箱)", Target: hybrid, Interaction: gc.InteractionTalk},
		}, actions)
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
		world.Components.Prop.Add(prop, &gc.Prop{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 30})
		world.Components.BlockPass.Add(prop, &gc.BlockPass{})
		world.Components.Interactable.Add(prop, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionMelee},
		})

		// 上に移動しようとする
		err := activity.ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 設置物に自動攻撃せず、移動もブロックされる
		grid := world.Components.GridElement.Get(player)
		assert.Equal(t, 10, int(grid.X))
		assert.Equal(t, 10, int(grid.Y))
		hp := world.Components.HP.Get(prop)
		assert.Equal(t, 30, hp.Current, "Propに自動攻撃しないのでHPは減らない")
	})
}

func TestGetSameTileManualActions(t *testing.T) {
	t.Parallel()

	t.Run("足元の手動の相互作用を出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnFieldItem(world, "healing_potion", 10, 10, 1)
		require.NoError(t, err)
		portal := world.ECS.NewEntity()
		world.Components.GridElement.Add(portal, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(portal, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionPortalNext},
		})

		assert.Equal(t, []InteractionAction{
			{Label: "Warp (next floor)", Target: portal, Interaction: gc.InteractionPortalNext},
		}, GetSameTileManualActions(world))
	})

	t.Run("別タイルのインタラクションは含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		portal := world.ECS.NewEntity()
		world.Components.GridElement.Add(portal, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.Interactable.Add(portal, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionPortalNext},
		})

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
		world.Components.Name.Add(e, &gc.Name{Name: i18n.Msgid(name)})
		// 熱量は材質×重量から導く。WOOD 200/kg × 50g = 10
		world.Components.Material.Add(e, &gc.Material{Kind: rawschema.WOOD})
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
		world.Components.Material.Add(f, &gc.Material{Kind: rawschema.WOOD})
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

// actionKinds は InteractionAction の種別だけを取り出す
func actionKinds(actions []InteractionAction) []gc.InteractionKind {
	kinds := make([]gc.InteractionKind, 0, len(actions))
	for _, a := range actions {
		kinds = append(kinds, a.Interaction)
	}
	return kinds
}

// TestGetInteractionActions_隣接のキューブはメニューだけを出す は、隣接の space メニューにキューブ
// メニューが出て、歩き込みで発動する入れ替えは出ないことを固定する。
func TestGetInteractionActions_隣接のキューブはメニューだけを出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 6, Y: 5})
	require.NoError(t, err)

	assert.Equal(t, []gc.InteractionKind{gc.InteractionOpenCubeMenu}, actionKinds(GetInteractionActions(world)))
}

func TestStartPickup(t *testing.T) {
	t.Parallel()

	t.Run("拾えるものが無ければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		trans, err := startPickup(world)
		require.NoError(t, err)

		assert.Equal(t, es.TransNone, trans.Type)
		assert.Nil(t, activity.GetLastResult(player, world))
	})

	t.Run("足元のアイテム1つはそのまま拾う", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		item, err := lifecycle.SpawnFieldItem(world, "healing_potion", 10, 10, 1)
		require.NoError(t, err)

		trans, err := startPickup(world)
		require.NoError(t, err)

		assert.Equal(t, es.TransNone, trans.Type)
		assert.True(t, world.Components.LocationInBackpack.Has(item))
	})

	t.Run("収納は1つでも拾うメニューで選ばせる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		crate, err := lifecycle.SpawnProp(world, "wooden_crate", 9, 9)
		require.NoError(t, err)

		trans, err := startPickup(world)
		require.NoError(t, err)

		assert.Equal(t, es.TransPush, trans.Type)
		assert.Equal(t, []InteractionAction{
			{Label: "Open Wooden Crate (upper left)", Target: crate, Interaction: gc.InteractionStorage},
		}, nearbyPickupActions(world))
	})

	t.Run("複数あれば拾うメニューを開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnFieldItem(world, "healing_potion", 10, 10, 1)
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
		require.NoError(t, err)

		trans, err := startPickup(world)
		require.NoError(t, err)

		assert.Equal(t, es.TransPush, trans.Type)
		assert.Nil(t, activity.GetLastResult(player, world))
	})
}

func TestPickupChoices(t *testing.T) {
	t.Parallel()

	t.Run("アイテムを拾っても残りがあれば閉じず、最後の1つで閉じる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		first, err := lifecycle.SpawnFieldItem(world, "healing_potion", 10, 10, 1)
		require.NoError(t, err)
		second, err := lifecycle.SpawnFieldItem(world, "bread", 10, 9, 1)
		require.NoError(t, err)

		_, choices := pickupChoices(world)
		require.Len(t, choices, 2)
		trans, err := choices[0].Run(world)
		require.NoError(t, err)
		assert.Equal(t, es.TransNone, trans.Type)

		_, choices = pickupChoices(world)
		require.Len(t, choices, 1)
		trans, err = choices[0].Run(world)
		require.NoError(t, err)
		assert.Equal(t, es.TransPop, trans.Type)

		assert.True(t, world.Components.LocationInBackpack.Has(first))
		assert.True(t, world.Components.LocationInBackpack.Has(second))
	})

	t.Run("収納は収納画面をメニューの上に開く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		_, err = lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
		require.NoError(t, err)

		_, choices := pickupChoices(world)
		require.Len(t, choices, 1)
		trans, err := choices[0].Run(world)
		require.NoError(t, err)

		assert.Equal(t, es.TransPush, trans.Type)
		require.Len(t, trans.NewStateFuncs, 1)
		state, err := trans.NewStateFuncs[0]()
		require.NoError(t, err)
		assert.IsType(t, &StorageMenuState{}, state)
	})
}

func TestStartPickup_周囲8タイルのアイテム1つはそのまま拾う(t *testing.T) {
	t.Parallel()
	for _, d := range []consts.Coord[consts.Tile]{
		{X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1},
		{X: -1, Y: 0}, {X: 1, Y: 0},
		{X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1},
	} {
		t.Run(fmt.Sprintf("dx=%d,dy=%d", d.X, d.Y), func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
			require.NoError(t, err)
			item, err := lifecycle.SpawnFieldItem(world, "healing_potion", 10+d.X, 10+d.Y, 1)
			require.NoError(t, err)

			trans, err := startPickup(world)
			require.NoError(t, err)

			assert.Equal(t, es.TransNone, trans.Type)
			assert.True(t, world.Components.LocationInBackpack.Has(item))
		})
	}
}

func TestNearbyPickupActions(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)
	_, err = lifecycle.SpawnFieldItem(world, "bread", 10, 10, 3)
	require.NoError(t, err)
	east, err := lifecycle.SpawnFieldItem(world, "healing_potion", 11, 10, 1)
	require.NoError(t, err)
	southEast, err := lifecycle.SpawnFieldItem(world, "healing_potion", 11, 11, 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnFieldItem(world, "healing_potion", 12, 10, 1)
	require.NoError(t, err)

	type row struct {
		Label       string
		Interaction gc.InteractionKind
	}
	actions := nearbyPickupActions(world)
	rows := make([]row, len(actions))
	for i, a := range actions {
		rows[i] = row{Label: a.Label, Interaction: a.Interaction}
	}

	assert.Equal(t, []row{
		{Label: "Open Wooden Crate (up)", Interaction: gc.InteractionStorage},
		{Label: "Pick up 3 Bread (here)", Interaction: gc.InteractionItem},
		{Label: "Pick up Healing Potion (right)", Interaction: gc.InteractionItem},
		{Label: "Pick up Healing Potion (lower right)", Interaction: gc.InteractionItem},
	}, rows)
	assert.Equal(t, crate, actions[0].Target)
	assert.Equal(t, east, actions[2].Target)
	assert.Equal(t, southEast, actions[3].Target)
}
