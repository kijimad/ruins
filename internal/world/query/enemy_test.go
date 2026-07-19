package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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
		world.Resources.SpriteSheets = spriteSheets

		// プレイヤーを配置
		playerEntity := world.ECS.NewEntity()
		world.Components.Player.Add(playerEntity, &gc.Player{})
		world.Components.GridElement.Add(playerEntity, &gc.GridElement{X: 10, Y: 10})

		// 視界内に敵を配置
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 12}, "火の玉")
		require.NoError(t, err)
		world.Components.Name.Set(enemy, &gc.Name{Name: "ゴブリン"})

		// 可視タイルに設定
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{
			{X: 12, Y: 12}: true,
		}

		enemies, err := query.GetVisibleEnemies(world)
		require.NoError(t, err)

		require.Len(t, enemies, 1, "視界内の敵が見つからない")

		// エンティティから情報を取得
		enemyEntity := enemies[0]
		name := query.GetEntityName(enemyEntity, world)
		grid := world.Components.GridElement.Get(enemyEntity)

		assert.Equal(t, "ゴブリン", name)
		assert.Equal(t, consts.Tile(12), grid.X)
		assert.Equal(t, consts.Tile(12), grid.Y)
	})

	t.Run("視界外の敵は取得されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.ECS.NewEntity()
		world.Components.Player.Add(playerEntity, &gc.Player{})
		world.Components.GridElement.Add(playerEntity, &gc.GridElement{X: 10, Y: 10})

		// 視界外に敵を配置（探索済みでない）
		_, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 50, Y: 50}, "火の玉")
		require.NoError(t, err)

		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

		enemies, err := query.GetVisibleEnemies(world)
		require.NoError(t, err)

		assert.Empty(t, enemies, "視界外の敵は取得されないべき")
	})

	t.Run("プレイヤーがいない場合は空を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーなし、敵のみ
		_, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "火の玉")
		require.NoError(t, err)

		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

		enemies, err := query.GetVisibleEnemies(world)

		require.Error(t, err, "プレイヤーがいない場合はエラーを返すべき")
		assert.Nil(t, enemies)
	})

	t.Run("SpawnEnemyで生成された敵は名前を持つ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.ECS.NewEntity()
		world.Components.Player.Add(playerEntity, &gc.Player{})
		world.Components.GridElement.Add(playerEntity, &gc.GridElement{X: 10, Y: 10})

		// 敵を配置
		_, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "火の玉")
		require.NoError(t, err)

		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{
			{X: 11, Y: 10}: true,
		}

		enemies, err := query.GetVisibleEnemies(world)
		require.NoError(t, err)

		require.Len(t, enemies, 1)

		// エンティティから名前を取得
		name := query.GetEntityName(enemies[0], world)
		assert.NotEmpty(t, name, "SpawnEnemyで生成された敵は名前を持つべき")
	})
}

func TestGetVisibleItems(t *testing.T) {
	t.Parallel()

	t.Run("視界内のアイテムを取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.ECS.NewEntity()
		world.Components.Player.Add(playerEntity, &gc.Player{})
		world.Components.GridElement.Add(playerEntity, &gc.GridElement{X: 10, Y: 10})

		// 視界内にアイテムを配置
		_, err := lifecycle.SpawnFieldItem(world, "回復薬", consts.Tile(12), consts.Tile(12), 1)
		require.NoError(t, err)

		// 可視タイルに設定
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{
			{X: 12, Y: 12}: true,
		}

		items, err := query.GetVisibleItems(world)
		require.NoError(t, err)

		require.Len(t, items, 1)

		// エンティティから情報を取得
		itemEntity := items[0]
		name := query.GetEntityName(itemEntity, world)
		grid := world.Components.GridElement.Get(itemEntity)
		desc := world.Components.Description.Get(itemEntity)

		assert.Equal(t, "回復薬", name)
		assert.NotNil(t, desc, "アイテムは説明を持つべき")
		if desc != nil {
			assert.NotEmpty(t, desc.Description)
		}
		assert.Equal(t, consts.Tile(12), grid.X)
		assert.Equal(t, consts.Tile(12), grid.Y)
	})

	t.Run("視界外のアイテムは取得されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.ECS.NewEntity()
		world.Components.Player.Add(playerEntity, &gc.Player{})
		world.Components.GridElement.Add(playerEntity, &gc.GridElement{X: 10, Y: 10})

		// 視界外にアイテムを配置
		_, err := lifecycle.SpawnFieldItem(world, "回復薬", consts.Tile(50), consts.Tile(50), 1)
		require.NoError(t, err)

		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

		items, err := query.GetVisibleItems(world)
		require.NoError(t, err)

		assert.Empty(t, items, "視界外のアイテムは取得されないべき")
	})

	t.Run("SpawnFieldItemで生成されたアイテムは名前を持つ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを配置
		playerEntity := world.ECS.NewEntity()
		world.Components.Player.Add(playerEntity, &gc.Player{})
		world.Components.GridElement.Add(playerEntity, &gc.GridElement{X: 10, Y: 10})

		// アイテムを配置
		_, err := lifecycle.SpawnFieldItem(world, "回復薬", consts.Tile(11), consts.Tile(10), 1)
		require.NoError(t, err)

		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{
			{X: 11, Y: 10}: true,
		}

		items, err := query.GetVisibleItems(world)
		require.NoError(t, err)

		require.Len(t, items, 1)

		// エンティティから名前を取得
		name := query.GetEntityName(items[0], world)
		assert.NotEmpty(t, name, "SpawnFieldItemで生成されたアイテムは名前を持つべき")
	})
}

func TestIsInVision(t *testing.T) {
	t.Parallel()

	t.Run("VisibleTilesがnilならfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		query.GetDungeon(world).VisibleTiles = nil

		assert.False(t, query.IsInVision(world, 0, 0, 5, 5))
	})

	t.Run("視界半径外は視界外と判定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{
			{X: 100, Y: 100}: true,
		}

		assert.False(t, query.IsInVision(world, 0, 0, 100, 100))
	})

	t.Run("VisibleTilesに含まれるタイルは視界内と判定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{
			{X: 5, Y: 5}: true,
		}

		assert.True(t, query.IsInVision(world, 0, 0, 5, 5))
	})

	t.Run("探索済みでもVisibleTilesに含まれないタイルは視界外と判定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		query.GetDungeon(world).ExploredTiles = map[gc.GridElement]bool{
			{X: 5, Y: 5}: true,
		}
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

		assert.False(t, query.IsInVision(world, 0, 0, 5, 5), "暗闘のタイルは探索済みでも視界外であるべき")
	})
}
