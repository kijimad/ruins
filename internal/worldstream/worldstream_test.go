package worldstream_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslateAllEntities(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: consts.Tile(100), TileHeight: consts.Tile(60)}))

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 20, Y: 15}, "fireball")
	require.NoError(t, err)

	// 西へ5・南へ2 平行移動（帯リベース相当）
	worldstream.TranslateAllEntities(world, -5, 2)

	pg := world.Components.GridElement.Get(player)
	assert.Equal(t, consts.Tile(5), pg.X, "プレイヤーX が dx ぶん移動する")
	assert.Equal(t, consts.Tile(12), pg.Y, "プレイヤーY が dy ぶん移動する")

	eg := world.Components.GridElement.Get(enemy)
	assert.Equal(t, consts.Tile(15), eg.X, "敵X も同じ dx で移動する")
	assert.Equal(t, consts.Tile(17), eg.Y, "敵Y も同じ dy で移動する")
}

func TestRemoveEntitiesInXRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: consts.Tile(100), TileHeight: consts.Tile(60)}))

	// プレイヤーは範囲内 [0,10) に居るが keep で残す
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 3, Y: 5}, "ash")
	require.NoError(t, err)
	// 範囲内の敵 → 削除される
	inside, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 2, Y: 5}, "fireball")
	require.NoError(t, err)
	// 範囲外の敵 → 残る
	outside, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 15, Y: 5}, "fireball")
	require.NoError(t, err)

	removed := worldstream.RemoveEntitiesInXRange(world, 0, 10, worldstream.KeepPlayer(world))

	assert.Equal(t, 1, removed, "範囲内の非keepエンティティ1体だけ削除される")
	assert.True(t, world.ECS.Alive(player), "プレイヤーは範囲内でも keep で残る")
	assert.False(t, world.ECS.Alive(inside), "範囲内の敵は削除される")
	assert.True(t, world.ECS.Alive(outside), "範囲外の敵は残る")
}

func TestRemoveEntitiesInXRange_境界は半開区間(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: consts.Tile(100), TileHeight: consts.Tile(60)}))
	if _, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 50, Y: 5}, "ash"); err != nil {
		require.NoError(t, err)
	}

	atLo, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 0, Y: 5}, "fireball") // lo は含む
	require.NoError(t, err)
	atHi, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 5}, "fireball") // hi は含まない
	require.NoError(t, err)

	removed := worldstream.RemoveEntitiesInXRange(world, 0, 10, nil)

	assert.Equal(t, 1, removed, "[0,10) の半開区間。X=0 は含み X=10 は含まない")
	assert.False(t, world.ECS.Alive(atLo), "X=lo は範囲内で削除")
	assert.True(t, world.ECS.Alive(atHi), "X=hi は範囲外で残る")
}

// TestRemoveEntitiesInXRange_所有者の収納在庫も道連れにする は、破棄される所有者が収納に持つ実体も
// 一緒に消えることを固定する。収納在庫は GridElement を持たず座標カリングでは拾われないため、
// 所有者だけ消えて在庫が孤児化するのを防ぐ。
func TestRemoveEntitiesInXRange_所有者の収納在庫も道連れにする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: consts.Tile(100), TileHeight: consts.Tile(60)}))

	// 所有者はフィールド上。在庫は GridElement を持たず収納にある
	owner := world.ECS.NewEntity()
	world.Components.GridElement.Add(owner, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 2, Y: 5}})
	stock, err := lifecycle.SpawnStorageItem(world, "wooden_sword", 1, owner)
	require.NoError(t, err)
	require.False(t, world.Components.GridElement.Has(stock), "在庫は座標を持たない")

	// 所有者を含む範囲を破棄する
	worldstream.RemoveEntitiesInXRange(world, 0, 10, nil)

	assert.False(t, world.ECS.Alive(owner), "所有者が消える")
	assert.False(t, world.ECS.Alive(stock), "収納在庫も道連れで消える")
}

// TestRemoveEntitiesInXRange_範囲外の所有者の在庫は残す は、破棄されない所有者の収納在庫は
// 消さないことを固定する。所有者ごとに在庫の可否を分ける。
func TestRemoveEntitiesInXRange_範囲外の所有者の在庫は残す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: consts.Tile(100), TileHeight: consts.Tile(60)}))

	owner := world.ECS.NewEntity()
	world.Components.GridElement.Add(owner, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 50, Y: 5}}) // 範囲外
	stock, err := lifecycle.SpawnStorageItem(world, "wooden_sword", 1, owner)
	require.NoError(t, err)

	worldstream.RemoveEntitiesInXRange(world, 0, 10, nil)

	assert.True(t, world.ECS.Alive(owner), "範囲外の所有者は残る")
	assert.True(t, world.ECS.Alive(stock), "範囲外の所有者の在庫も残る")
}
