package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager
		world.Resources.Dungeon.Level.TileWidth = 50
		world.Resources.Dungeon.Level.TileHeight = 50

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		// 移動を実行
		var history []HistoryEntry
		manager.History = &history
		assert.NoError(t, ExecuteMoveAction(world, gc.DirectionUp))

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
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		assert.Error(t, ExecuteMoveAction(world, gc.DirectionUp))
	})

	t.Run("GridElementがない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		assert.NoError(t, ExecuteMoveAction(world, gc.DirectionUp))
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
				world.Resources.TurnManager = turns.NewTurnManager()
				manager := NewManager(nil)
				world.Resources.ActivityManager = manager
				world.Resources.Dungeon.Level.TileWidth = 50
				world.Resources.Dungeon.Level.TileHeight = 50

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
		world.Resources.TurnManager = turns.NewTurnManager()
		world.Config.RNG = rand.New(rand.NewPCG(42, 0))
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		initialEnemyHP := enemyPools.HP.Current

		// 移動（攻撃）を実行
		var history []HistoryEntry
		manager.History = &history
		err = ExecuteMoveAction(world, gc.DirectionUp)
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
}

func TestExecuteWaitAction(t *testing.T) {
	t.Parallel()

	t.Run("待機アクションの実行", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{})

		var history []HistoryEntry
		manager.History = &history
		assert.NoError(t, ExecuteWaitAction(world))

		require.Len(t, history, 1)
		assert.Equal(t, "Wait", history[0].Activity.String())
		assert.True(t, history[0].Success)
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		assert.Error(t, ExecuteWaitAction(world))
	})
}

func TestExecuteEnterAction(t *testing.T) {
	t.Parallel()

	t.Run("何もない場所でEnter", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		var history []HistoryEntry
		manager.History = &history
		assert.NoError(t, ExecuteEnterAction(world))

		// 何も実行されない
		assert.Len(t, history, 0)
	})

	t.Run("アイテムがある場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

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

		var history []HistoryEntry
		manager.History = &history
		// Pickupは検証失敗する（拾えるアイテムがない）がエラーは返さない
		err := ExecuteEnterAction(world)

		// Pickupが試行される（検証失敗してもエラーは返さない設計）
		require.Len(t, history, 1)
		assert.Equal(t, "Pickup", history[0].Activity.String())
		// 検証失敗のため成功フラグはfalse
		assert.False(t, history[0].Success)
		assert.Error(t, err)
	})

	t.Run("プレイヤーが存在しない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		assert.Error(t, ExecuteEnterAction(world))
	})

	t.Run("GridElementがない場合", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.TurnManager = turns.NewTurnManager()
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		assert.NoError(t, ExecuteEnterAction(world))
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

func TestCheckTileEvents(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーエンティティの場合のみイベントチェック", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// パニックしないことを確認
		checkTileEvents(world, player, 10, 10)
	})

	t.Run("非プレイヤーエンティティの場合はイベントチェックしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// パニックしないことを確認
		checkTileEvents(world, enemy, 10, 10)
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
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		_, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemy.AddComponent(world.Components.Dead, &gc.Dead{})

		// 移動を実行
		var history []HistoryEntry
		manager.History = &history
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)

		// 検証: 攻撃ではなく移動になる
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
		manager := NewManager(nil)
		world.Resources.ActivityManager = manager

		_, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)
		enemy, err := worldhelper.SpawnEnemy(world, 10, 9, "火の玉")
		require.NoError(t, err)
		enemyPools := world.Components.Pools.Get(enemy).(*gc.Pools)
		enemyPools.HP.Current = 1

		var history []HistoryEntry
		manager.History = &history

		// 1回目: 攻撃で敵を倒す
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		assert.True(t, enemy.HasComponent(world.Components.Dead))
		require.Len(t, history, 1)
		assert.Equal(t, "Attack", history[0].Activity.String())

		// 2回目: 死亡した敵がいた場所への移動
		err = ExecuteMoveAction(world, gc.DirectionUp)
		require.NoError(t, err)
		require.Len(t, history, 2)
		assert.Equal(t, "Move", history[1].Activity.String())
		assert.True(t, history[1].Success)
	})
}
