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

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 20, Y: 15}, "火の玉")
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
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 3, Y: 5}, "Ash")
	require.NoError(t, err)
	// 範囲内の敵 → 削除される
	inside, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 2, Y: 5}, "火の玉")
	require.NoError(t, err)
	// 範囲外の敵 → 残る
	outside, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 15, Y: 5}, "火の玉")
	require.NoError(t, err)

	removed := worldstream.RemoveEntitiesInXRange(world, 0, 10, worldstream.KeepPlayerAndSquad(world))

	assert.Equal(t, 1, removed, "範囲内の非keepエンティティ1体だけ削除される")
	assert.True(t, world.ECS.Alive(player), "プレイヤーは範囲内でも keep で残る")
	assert.False(t, world.ECS.Alive(inside), "範囲内の敵は削除される")
	assert.True(t, world.ECS.Alive(outside), "範囲外の敵は残る")
}

func TestRemoveEntitiesInXRange_境界は半開区間(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: consts.Tile(100), TileHeight: consts.Tile(60)}))
	if _, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 50, Y: 5}, "Ash"); err != nil {
		require.NoError(t, err)
	}

	atLo, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 0, Y: 5}, "火の玉") // lo は含む
	require.NoError(t, err)
	atHi, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 5}, "火の玉") // hi は含まない
	require.NoError(t, err)

	removed := worldstream.RemoveEntitiesInXRange(world, 0, 10, nil)

	assert.Equal(t, 1, removed, "[0,10) の半開区間。X=0 は含み X=10 は含まない")
	assert.False(t, world.ECS.Alive(atLo), "X=lo は範囲内で削除")
	assert.True(t, world.ECS.Alive(atHi), "X=hi は範囲外で残る")
}
