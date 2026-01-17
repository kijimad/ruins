package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/turns"
	"github.com/stretchr/testify/assert"
)

func TestExecuteMoveAction(t *testing.T) {
	t.Parallel()

	t.Run("正常な移動", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

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
		ExecuteMoveAction(world, gc.DirectionUp)

		// 移動後の座標を確認
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, initialX, int(gridAfter.X), "X座標は変化しないべき")
		assert.Equal(t, initialY-1, int(gridAfter.Y), "Y座標が1減るべき")
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしで移動を試みる（パニックしないことを確認）
		ExecuteMoveAction(world, gc.DirectionUp)
		// エラーにならず何も起きないべき
	})

	t.Run("GridElementがない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// 移動を試みる（パニックしないことを確認）
		ExecuteMoveAction(world, gc.DirectionUp)
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

				player := world.Manager.NewEntity()
				player.AddComponent(world.Components.Player, &gc.Player{})
				player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
				player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

				ExecuteMoveAction(world, tt.direction)

				gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
				assert.Equal(t, tt.expectedX, int(gridAfter.X), "X座標が正しく移動するべき")
				assert.Equal(t, tt.expectedY, int(gridAfter.Y), "Y座標が正しく移動するべき")
			})
		}
	})
}

func TestExecuteWaitAction(t *testing.T) {
	t.Parallel()

	t.Run("待機アクションの実行", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 待機アクションを実行（パニックしないことを確認）
		ExecuteWaitAction(world)

		// プレイヤーエンティティが存在することを確認
		assert.True(t, player.HasComponent(world.Components.Player), "プレイヤーが存在するべき")
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしで待機を試みる（パニックしないことを確認）
		ExecuteWaitAction(world)
		// エラーにならず何も起きないべき
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
		ExecuteEnterAction(world)

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

		// 同じ位置にワープホールを作成
		warp := world.Manager.NewEntity()
		warp.AddComponent(world.Components.Interactable, &gc.Interactable{Data: gc.WarpNextInteraction{}})
		warp.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// Enterアクションを実行（ワープ処理が呼ばれることを期待）
		ExecuteEnterAction(world)

		// ワープ処理が実行されたかは、実装によって検証方法が異なる
		// ここではパニックしないことを確認
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなしでEnterを試みる（パニックしないことを確認）
		ExecuteEnterAction(world)
		// エラーにならず何も起きないべき
	})

	t.Run("GridElementがない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// GridElementなしのプレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// Enterを試みる（パニックしないことを確認）
		ExecuteEnterAction(world)
		// エラーにならず何も起きないべき
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
		ExecuteEnterAction(world)

		// 何も起きないことを確認（パニックしない）
		assert.True(t, player.HasComponent(world.Components.Player), "プレイヤーが存在するべき")
	})
}

func TestExecuteMoveActionWithEnemy(t *testing.T) {
	t.Parallel()

	t.Run("敵がいる位置への移動は攻撃になる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// ターンマネージャーを初期化
		turnManager := turns.NewTurnManager()
		world.Resources.TurnManager = turnManager

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.FactionAlly, &gc.FactionAlly)
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 北隣に敵を作成
		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 9})
		enemy.AddComponent(world.Components.TurnBased, &gc.TurnBased{})
		enemy.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 100, Max: 100},
		})
		enemy.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{},
		})

		initialPlayerX := int(world.Components.GridElement.Get(player).(*gc.GridElement).X)
		initialPlayerY := int(world.Components.GridElement.Get(player).(*gc.GridElement).Y)

		// 北に移動（敵がいる方向）
		ExecuteMoveAction(world, gc.DirectionUp)

		// プレイヤーが移動していないことを確認（攻撃したため）
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, initialPlayerX, int(gridAfter.X), "攻撃時はX座標が変化しないべき")
		assert.Equal(t, initialPlayerY, int(gridAfter.Y), "攻撃時はY座標が変化しないべき")

		// 敵エンティティが存在することを確認（攻撃が実行された）
		assert.True(t, enemy.HasComponent(world.Components.Pools), "敵が存在するべき")
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
