package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVisibleEnemies(t *testing.T) {
	t.Parallel()

	t.Run("視界内の敵を取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// SpriteSheetsを初期化
		spriteSheets := make(map[string]gc.SpriteSheet)
		spriteSheets["field"] = gc.SpriteSheet{
			Sprites: map[string]gc.Sprite{
				"red_ball": {Width: 32, Height: 32},
			},
		}
		world.Resources.SpriteSheets = &spriteSheets

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 視界内に敵を配置
		enemy, err := SpawnEnemy(world, 12, 12, "火の玉")
		require.NoError(t, err)
		enemy.AddComponent(world.Components.Name, &gc.Name{Name: "ゴブリン"})

		// 探索済みタイルに設定
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 12, Y: 12}] = true

		enemies, err := GetVisibleEnemies(world)
		require.NoError(t, err)

		require.Len(t, enemies, 1, "視界内の敵が見つからない")
		assert.Equal(t, "ゴブリン", enemies[0].Name)
		assert.Equal(t, 12, enemies[0].GridX)
		assert.Equal(t, 12, enemies[0].GridY)
	})

	t.Run("視界外の敵は取得されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 視界外に敵を配置（探索済みでない）
		enemy, err := SpawnEnemy(world, 50, 50, "火の玉")
		require.NoError(t, err)
		_ = enemy

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)

		enemies, err := GetVisibleEnemies(world)
		require.NoError(t, err)

		assert.Empty(t, enemies, "視界外の敵は取得されないべき")
	})

	t.Run("複数の敵を距離順にソートして取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 遠い敵
		enemy1, err := SpawnEnemy(world, 15, 10, "火の玉")
		require.NoError(t, err)
		enemy1.AddComponent(world.Components.Name, &gc.Name{Name: "遠い敵"})

		// 近い敵
		enemy2, err := SpawnEnemy(world, 11, 10, "火の玉")
		require.NoError(t, err)
		enemy2.AddComponent(world.Components.Name, &gc.Name{Name: "近い敵"})

		// 探索済みタイルに設定
		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 11, Y: 10}] = true
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 15, Y: 10}] = true

		enemies, err := GetVisibleEnemies(world)
		require.NoError(t, err)

		require.Len(t, enemies, 2)
		assert.Equal(t, "近い敵", enemies[0].Name, "最初は近い敵であるべき")
		assert.Equal(t, "遠い敵", enemies[1].Name, "次は遠い敵であるべき")
		assert.Less(t, enemies[0].Distance, enemies[1].Distance, "距離順にソートされているべき")
	})

	t.Run("プレイヤーがいない場合は空を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなし、敵のみ
		enemy, err := SpawnEnemy(world, 5, 5, "火の玉")
		require.NoError(t, err)
		_ = enemy

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)

		enemies, err := GetVisibleEnemies(world)

		assert.Error(t, err, "プレイヤーがいない場合はエラーを返すべき")
		assert.Nil(t, enemies)
	})

	t.Run("SpawnEnemyで生成された敵は名前を持つ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 敵を配置
		_, err := SpawnEnemy(world, 11, 10, "火の玉")
		require.NoError(t, err)

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 11, Y: 10}] = true

		enemies, err := GetVisibleEnemies(world)
		require.NoError(t, err)

		require.Len(t, enemies, 1)
		assert.NotEmpty(t, enemies[0].Name, "SpawnEnemyで生成された敵は名前を持つべき")
	})
}

func TestGetVisibleItems(t *testing.T) {
	t.Parallel()

	t.Run("視界内のアイテムを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 視界内にアイテムを配置
		_, err := SpawnFieldItem(world, "回復薬", gc.Tile(12), gc.Tile(12))
		require.NoError(t, err)

		// 探索済みタイルに設定
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 12, Y: 12}] = true

		items, err := GetVisibleItems(world)
		require.NoError(t, err)

		require.Len(t, items, 1)
		assert.Equal(t, "回復薬", items[0].Name)
		assert.NotEmpty(t, items[0].Description, "アイテムは説明を持つべき")
		assert.Equal(t, 12, items[0].GridX)
		assert.Equal(t, 12, items[0].GridY)
	})

	t.Run("視界外のアイテムは取得されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 視界外にアイテムを配置
		_, err := SpawnFieldItem(world, "回復薬", gc.Tile(50), gc.Tile(50))
		require.NoError(t, err)

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)

		items, err := GetVisibleItems(world)
		require.NoError(t, err)

		assert.Empty(t, items, "視界外のアイテムは取得されないべき")
	})

	t.Run("複数のアイテムを距離順にソートして取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 遠いアイテム
		item1, err := SpawnFieldItem(world, "回復薬", gc.Tile(15), gc.Tile(10))
		require.NoError(t, err)
		item1.AddComponent(world.Components.Name, &gc.Name{Name: "遠いアイテム"})

		// 近いアイテム
		item2, err2 := SpawnFieldItem(world, "回復薬", gc.Tile(11), gc.Tile(10))
		require.NoError(t, err2)
		item2.AddComponent(world.Components.Name, &gc.Name{Name: "近いアイテム"})

		// 探索済みタイルに設定
		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 11, Y: 10}] = true
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 15, Y: 10}] = true

		items, err := GetVisibleItems(world)
		require.NoError(t, err)

		require.Len(t, items, 2)
		assert.Equal(t, "近いアイテム", items[0].Name, "最初は近いアイテムであるべき")
		assert.Equal(t, "遠いアイテム", items[1].Name, "次は遠いアイテムであるべき")
		assert.Less(t, items[0].Distance, items[1].Distance, "距離順にソートされているべき")
	})

	t.Run("SpawnFieldItemで生成されたアイテムは名前を持つ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// アイテムを配置
		_, err := SpawnFieldItem(world, "回復薬", gc.Tile(11), gc.Tile(10))
		require.NoError(t, err)

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 11, Y: 10}] = true

		items, err := GetVisibleItems(world)
		require.NoError(t, err)

		require.Len(t, items, 1)
		assert.NotEmpty(t, items[0].Name, "SpawnFieldItemで生成されたアイテムは名前を持つべき")
	})
}

func TestIsInVision(t *testing.T) {
	t.Parallel()

	t.Run("探索済みタイルは視界内と判定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 5, Y: 5}] = true

		result := isInVision(world, 0, 0, 5, 5)

		assert.True(t, result, "探索済みタイルは視界内であるべき")
	})

	t.Run("探索済みでないタイルは視界外と判定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)

		result := isInVision(world, 0, 0, 5, 5)

		assert.False(t, result, "探索済みでないタイルは視界外であるべき")
	})

	t.Run("視界半径外は視界外と判定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)
		world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 100, Y: 100}] = true

		result := isInVision(world, 0, 0, 100, 100)

		assert.False(t, result, "視界半径外は視界外であるべき")
	})
}
