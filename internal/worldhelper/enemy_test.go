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

		// エンティティから情報を取得
		enemyEntity := enemies[0]
		name := GetEntityName(enemyEntity, world)
		grid := world.Components.GridElement.Get(enemyEntity).(*gc.GridElement)

		assert.Equal(t, "ゴブリン", name)
		assert.Equal(t, gc.Tile(12), grid.X)
		assert.Equal(t, gc.Tile(12), grid.Y)
	})

	t.Run("視界外の敵は取得されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.Manager.NewEntity()
		playerEntity.AddComponent(world.Components.Player, &gc.Player{})
		playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

		// 視界外に敵を配置（探索済みでない）
		_, err := SpawnEnemy(world, 50, 50, "火の玉")
		require.NoError(t, err)

		world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)

		enemies, err := GetVisibleEnemies(world)
		require.NoError(t, err)

		assert.Empty(t, enemies, "視界外の敵は取得されないべき")
	})

	t.Run("プレイヤーがいない場合は空を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなし、敵のみ
		_, err := SpawnEnemy(world, 5, 5, "火の玉")
		require.NoError(t, err)

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

		// エンティティから名前を取得
		name := GetEntityName(enemies[0], world)
		assert.NotEmpty(t, name, "SpawnEnemyで生成された敵は名前を持つべき")
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

		// エンティティから情報を取得
		itemEntity := items[0]
		name := GetEntityName(itemEntity, world)
		grid := world.Components.GridElement.Get(itemEntity).(*gc.GridElement)
		desc := world.Components.Description.Get(itemEntity)

		assert.Equal(t, "回復薬", name)
		assert.NotNil(t, desc, "アイテムは説明を持つべき")
		if desc != nil {
			assert.NotEmpty(t, desc.(*gc.Description).Description)
		}
		assert.Equal(t, gc.Tile(12), grid.X)
		assert.Equal(t, gc.Tile(12), grid.Y)
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

		// エンティティから名前を取得
		name := GetEntityName(items[0], world)
		assert.NotEmpty(t, name, "SpawnFieldItemで生成されたアイテムは名前を持つべき")
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
