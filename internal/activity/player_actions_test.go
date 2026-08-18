package activity

import (
	"math/rand/v2"
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteMoveAction(t *testing.T) {
	t.Parallel()

	t.Run("正常な移動", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 移動を実行
		require.NoError(t, ExecuteMoveAction(world, gc.DirectionUp))

		// 検証
		result := GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMove, result.BehaviorName)
		assert.True(t, result.Success)
		gridAfter := world.Components.GridElement.Get(player)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 9, int(gridAfter.Y))
	})

	t.Run("重量超過では移動せず致命エラーにもならない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.TurnBased.Add(player, &gc.TurnBased{})
		// 最大の1.5倍を超える積載。移動は Validate で弾かれる
		world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{Max: 100, Current: 200})

		// 重すぎて動けないのは通常の状態。入力層へエラーを返さず no-op にする
		require.NoError(t, ExecuteMoveAction(world, gc.DirectionUp))

		grid := world.Components.GridElement.Get(player)
		assert.Equal(t, 10, int(grid.X), "移動していない")
		assert.Equal(t, 10, int(grid.Y), "移動していない")
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		assert.Error(t, ExecuteMoveAction(world, gc.DirectionUp))
	})

	t.Run("GridElementがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

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

				player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
				require.NoError(t, err)

				require.NoError(t, ExecuteMoveAction(world, tt.direction))

				grid := world.Components.GridElement.Get(player)
				assert.Equal(t, tt.expectedX, int(grid.X))
				assert.Equal(t, tt.expectedY, int(grid.Y))
			})
		}
	})

	t.Run("敵がいる位置への移動は攻撃になる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		enemyHP := world.Components.HP.Get(enemy)
		initialEnemyHP := enemyHP.Current

		// 移動（攻撃）を実行
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: Attackが実行される
		result := GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)
		assert.True(t, result.Success)
		gridAfter := world.Components.GridElement.Get(player)
		assert.Equal(t, 10, int(gridAfter.X))
		assert.Equal(t, 10, int(gridAfter.Y))
		assert.Less(t, enemyHP.Current, initialEnemyHP)
	})
}

func TestShowTileInteractionMessage_床の同種スタックは1行にまとめる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// タイル(10,10)に同種を3個置く
	_, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 10, 10, 3)
	require.NoError(t, err)

	playerGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}
	showTileInteractionMessage(world, playerGrid)

	hits := 0
	for _, e := range query.GetGameLog(world).GetRecentEntries(10) {
		if strings.Contains(e.Text(), "is here") {
			hits++
		}
	}
	assert.Equal(t, 1, hits, "同種3個でもログは1行にまとまる")
}

func TestExecuteWaitAction(t *testing.T) {
	t.Parallel()

	t.Run("待機アクションの実行", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		require.NoError(t, ExecuteWaitAction(world))

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

func TestInteractablesAtSameTile(t *testing.T) {
	t.Parallel()

	t.Run("同一タイルに複数あれば全件返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		coord := consts.Coord[consts.Tile]{X: 10, Y: 10}

		// ポータルと NPC が同じタイルに同居する。先着1件でなく両方返ることを固定する。
		portal := world.ECS.NewEntity()
		world.Components.GridElement.Add(portal, &gc.GridElement{Coord: coord})
		world.Components.Interactable.Add(portal, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionPortalNext},
		})
		npc := world.ECS.NewEntity()
		world.Components.GridElement.Add(npc, &gc.GridElement{Coord: coord})
		world.Components.Interactable.Add(npc, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionTalk},
		})

		targetGrid := &gc.GridElement{Coord: coord}
		found := interactablesAtSameTile(world, targetGrid)

		require.Len(t, found, 2)
		assert.Contains(t, found, portal)
		assert.Contains(t, found, npc)
	})

	t.Run("同じタイルのInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Interactableエンティティを作成
		interactableEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(interactableEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(interactableEntity, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})

		targetGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}
		found := interactablesAtSameTile(world, targetGrid)

		require.Len(t, found, 1)
		assert.Equal(t, interactableEntity, found[0])
	})

	t.Run("異なるタイルのInteractableは取得されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Interactableエンティティを作成（異なる位置）
		interactableEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(interactableEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 15, Y: 15}})
		world.Components.Interactable.Add(interactableEntity, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})

		targetGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}
		found := interactablesAtSameTile(world, targetGrid)

		assert.Empty(t, found)
	})

	t.Run("死亡エンティティはInteractable対象から除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 死亡したInteractableエンティティを作成
		deadEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(deadEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(deadEntity, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem},
		})
		world.Components.Dead.Add(deadEntity, &gc.Dead{})

		targetGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}
		found := interactablesAtSameTile(world, targetGrid)

		assert.Empty(t, found)
	})
}

func TestGetAllInteractiveInteractablesInRange(t *testing.T) {
	t.Parallel()

	t.Run("Manual方式のInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Manual方式のInteractableを作成
		manualEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(manualEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Interactable.Add(manualEntity, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionItem}, // Manual + SameTile
		})

		targetGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}
		results := GetAllInteractiveInteractablesInRange(world, targetGrid)

		require.Len(t, results, 1)
		assert.Equal(t, manualEntity, results[0])
	})

	t.Run("OnCollision方式のInteractableを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// OnCollision方式のInteractableを作成
		collisionEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(collisionEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})
		world.Components.Interactable.Add(collisionEntity, &gc.Interactable{
			Interactions: []gc.InteractionKind{gc.InteractionMelee}, // OnCollision + Adjacent
		})

		targetGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}
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
		{"直上", 10, 10, 10, 10, "here"},
		{"上", 10, 10, 10, 9, "up"},
		{"下", 10, 10, 10, 11, "down"},
		{"左", 10, 10, 9, 10, "left"},
		{"右", 10, 10, 11, 10, "right"},
		{"左上", 10, 10, 9, 9, "upper left"},
		{"右上", 10, 10, 11, 9, "upper right"},
		{"左下", 10, 10, 9, 11, "lower left"},
		{"右下", 10, 10, 11, 11, "lower right"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			playerGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(tt.playerX), Y: consts.Tile(tt.playerY)}}
			targetGrid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(tt.targetX), Y: consts.Tile(tt.targetY)}}

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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		world.Components.Dead.Add(enemy, &gc.Dead{})

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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 9}, "fireball")
		require.NoError(t, err)
		enemyHP := world.Components.HP.Get(enemy)
		enemyHP.Current = 1

		// 1回目: 攻撃で敵を倒す
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		assert.True(t, world.Components.Dead.Has(enemy))
		result := GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMelee, result.BehaviorName)

		// 2回目: 死亡した敵がいた場所への移動
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		result = GetLastResult(player, world)
		require.NotNil(t, result)
		assert.Equal(t, gc.BehaviorMove, result.BehaviorName)
		assert.True(t, result.Success)
	})
}
