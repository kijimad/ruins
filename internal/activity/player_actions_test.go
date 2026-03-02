package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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
		assert.NoError(t, ExecuteMoveAction(world, gc.DirectionUp))

		// 検証
		result := GetLastResult(player, world)
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

		assert.Error(t, ExecuteMoveAction(world, gc.DirectionUp))
	})

	t.Run("GridElementがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		assert.Error(t, ExecuteMoveAction(world, gc.DirectionUp))
	})

	t.Run("8方向の移動", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
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

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				world := testutil.InitTestWorld(t)

				player := world.Manager.NewEntity()
				player.AddComponent(world.Components.Player, &gc.Player{})
				player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
				player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

				assert.NoError(t, ExecuteMoveAction(world, tt.direction))

				grid := world.Components.GridElement.Get(player).(*gc.GridElement)
				assert.Equal(t, tt.expectedX, int(grid.X))
				assert.Equal(t, tt.expectedY, int(grid.Y))
			})
		}
	})

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
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorAttack, result.BehaviorName)
		assert.True(t, result.Success)
		gridAfter := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 10, int(gridAfter.Y))
		assert.Less(t, enemyPools.HP.Current, initialEnemyHP)
	})
}

func TestExecuteWaitAction(t *testing.T) {
	t.Parallel()

	t.Run("待機アクションの実行", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		assert.NoError(t, ExecuteWaitAction(world))

		result := GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorWait, result.BehaviorName)
		assert.True(t, result.Success)
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		assert.Error(t, ExecuteWaitAction(world))
	})
}

func TestExecuteEnterAction(t *testing.T) {
	t.Parallel()

	t.Run("何もない場所でEnter", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		assert.NoError(t, ExecuteEnterAction(world))

		// 何も実行されないのでLastActivityはnil
		result := GetLastResult(player, world)
		assert.Nil(t, result)
	})

	t.Run("アイテムがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// アイテムを作成（同タイル、手動発動）
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		item.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.ItemInteraction{},
		})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "テストアイテム"})
		item.AddComponent(world.Components.Item, &gc.Item{})

		// ItemLocationOnFieldがないので拾えるアイテムがない
		err := ExecuteEnterAction(world)
		assert.Error(t, err)

		// Pickupが試行され、検証失敗でエラー
		result := GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorPickup, result.BehaviorName)
		assert.False(t, result.Success)
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		assert.Error(t, ExecuteEnterAction(world))
	})

	t.Run("GridElementがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		assert.Error(t, ExecuteEnterAction(world))
	})
}

func TestGetInteractableAtSameTile(t *testing.T) {
	t.Parallel()

	t.Run("同じタイルのInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Interactableエンティティを作成
		interactableEntity := world.Manager.NewEntity()
		interactableEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		interactableEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.ItemInteraction{},
		})

		targetGrid := &gc.GridElement{X: 10, Y: 10}
		interactable, entity := getInteractableAtSameTile(world, targetGrid)

		require.NotNil(t, interactable)
		assert.Equal(t, interactableEntity, entity)
	})

	t.Run("異なるタイルのInteractableは取得されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Interactableエンティティを作成（異なる位置）
		interactableEntity := world.Manager.NewEntity()
		interactableEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 15, Y: 15})
		interactableEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.ItemInteraction{},
		})

		targetGrid := &gc.GridElement{X: 10, Y: 10}
		interactable, _ := getInteractableAtSameTile(world, targetGrid)

		assert.Nil(t, interactable)
	})

	t.Run("死亡エンティティはInteractable対象から除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 死亡したInteractableエンティティを作成
		deadEntity := world.Manager.NewEntity()
		deadEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		deadEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.ItemInteraction{},
		})
		deadEntity.AddComponent(world.Components.Dead, &gc.Dead{})

		targetGrid := &gc.GridElement{X: 10, Y: 10}
		interactable, _ := getInteractableAtSameTile(world, targetGrid)

		assert.Nil(t, interactable)
	})
}

func TestGetInteractableInRange(t *testing.T) {
	t.Parallel()

	t.Run("範囲内のInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 隣接範囲のInteractableを作成
		interactableEntity := world.Manager.NewEntity()
		interactableEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		interactableEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{}, // Adjacent範囲
		})

		targetGrid := &gc.GridElement{X: 10, Y: 10}
		interactable, entity := getInteractableInRange(world, targetGrid)

		require.NotNil(t, interactable)
		assert.Equal(t, interactableEntity, entity)
	})

	t.Run("死亡エンティティは範囲内でも除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 死亡したInteractableを作成
		deadEntity := world.Manager.NewEntity()
		deadEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		deadEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{},
		})
		deadEntity.AddComponent(world.Components.Dead, &gc.Dead{})

		targetGrid := &gc.GridElement{X: 10, Y: 10}
		interactable, _ := getInteractableInRange(world, targetGrid)

		assert.Nil(t, interactable)
	})
}

func TestGetAllInteractiveInteractablesInRange(t *testing.T) {
	t.Parallel()

	t.Run("Manual方式のInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Manual方式のInteractableを作成
		manualEntity := world.Manager.NewEntity()
		manualEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		manualEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.ItemInteraction{}, // Manual + SameTile
		})

		targetGrid := &gc.GridElement{X: 10, Y: 10}
		results := GetAllInteractiveInteractablesInRange(world, targetGrid)

		require.Len(t, results, 1)
		assert.Equal(t, manualEntity, results[0])
	})

	t.Run("OnCollision方式のInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// OnCollision方式のInteractableを作成
		collisionEntity := world.Manager.NewEntity()
		collisionEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
		collisionEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
			Data: gc.MeleeInteraction{}, // OnCollision + Adjacent
		})

		targetGrid := &gc.GridElement{X: 10, Y: 10}
		results := GetAllInteractiveInteractablesInRange(world, targetGrid)

		require.Len(t, results, 1)
		assert.Equal(t, collisionEntity, results[0])
	})
}

func TestGetDirectionLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		playerX  int
		playerY  int
		targetX  int
		targetY  int
		expected string
	}{
		{"直上", 10, 10, 10, 10, "直上"},
		{"上", 10, 10, 10, 9, "上"},
		{"下", 10, 10, 10, 11, "下"},
		{"左", 10, 10, 9, 10, "左"},
		{"右", 10, 10, 11, 10, "右"},
		{"左上", 10, 10, 9, 9, "左上"},
		{"右上", 10, 10, 11, 9, "右上"},
		{"左下", 10, 10, 9, 11, "左下"},
		{"右下", 10, 10, 11, 11, "右下"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			playerGrid := &gc.GridElement{X: consts.Tile(tt.playerX), Y: consts.Tile(tt.playerY)}
			targetGrid := &gc.GridElement{X: consts.Tile(tt.targetX), Y: consts.Tile(tt.targetY)}

			result := GetDirectionLabel(playerGrid, targetGrid)
			assert.Equal(t, tt.expected, result)
		})
	}
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
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: 攻撃ではなく移動になる
		result := GetLastResult(player, world)
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
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		assert.True(t, enemy.HasComponent(world.Components.Dead))
		result := GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorAttack, result.BehaviorName)

		// 2回目: 死亡した敵がいた場所への移動
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		result = GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMove, result.BehaviorName)
		assert.True(t, result.Success)
	})
}
