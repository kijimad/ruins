package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countGridEntitiesAt は指定座標にあるエンティティ数を数える。
func countGridEntitiesAt(t *testing.T, world w.World, coord consts.Coord[consts.Tile]) int {
	t.Helper()
	n := 0
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		if world.Components.GridElement.Get(q.Entity()).Coord == coord {
			n++
		}
	}
	return n
}

// TestSpawnPropNearPlayer はプレイヤーの近傍に prop を生成すること、プレイヤー不在では
// エラーを返すことを検証する。
func TestSpawnPropNearPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	target := consts.Coord[consts.Tile]{X: 7, Y: 5} // playerX+2
	before := countGridEntitiesAt(t, world, target)
	require.NoError(t, spawnPropNearPlayer(world, "barrel"))
	assert.Equal(t, before+1, countGridEntitiesAt(t, world, target), "近傍に prop を1体生成する")

	empty := testutil.InitTestWorld(t)
	assert.Error(t, spawnPropNearPlayer(empty, "barrel"))
}

// TestSpawnEnemyNearPlayer はプレイヤーの近傍に敵を生成すること、プレイヤー不在では
// エラーを返すことを検証する。
func TestSpawnEnemyNearPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	target := consts.Coord[consts.Tile]{X: 13, Y: 5} // playerX+8
	require.NoError(t, spawnEnemyNearPlayer(world, "moss_turtle"))
	assert.Equal(t, 1, countGridEntitiesAt(t, world, target), "近傍に敵を1体生成する")

	empty := testutil.InitTestWorld(t)
	assert.Error(t, spawnEnemyNearPlayer(empty, "moss_turtle"))
}

// TestSpawnStorageWithItems は収納 prop とその在庫を生成すること、プレイヤー不在では
// エラーを返すことを検証する。
func TestSpawnStorageWithItems(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	target := consts.Coord[consts.Tile]{X: 7, Y: 5} // playerX+2
	before := countGridEntitiesAt(t, world, target)
	require.NoError(t, spawnStorageWithItems(world))
	assert.Equal(t, before+1, countGridEntitiesAt(t, world, target), "収納 prop を1体生成する")

	empty := testutil.InitTestWorld(t)
	assert.Error(t, spawnStorageWithItems(empty))
}
