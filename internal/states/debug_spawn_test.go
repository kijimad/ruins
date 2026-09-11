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
	defer q.Close() // 早期リターンを入れても world をアンロックする安全ネット
	for q.Next() {
		if world.Components.GridElement.Get(q.Entity()).Coord == coord {
			n++
		}
	}
	return n
}

func TestSpawnPropNearPlayer(t *testing.T) {
	t.Parallel()

	t.Run("近傍にpropを生成する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)

		target := consts.Coord[consts.Tile]{X: 7, Y: 5} // playerX+2
		before := countGridEntitiesAt(t, world, target)
		require.NoError(t, spawnPropNearPlayer(world, "barrel"))
		assert.Equal(t, before+1, countGridEntitiesAt(t, world, target), "近傍に prop を1体生成する")
	})

	t.Run("プレイヤー不在はエラーを返す", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, spawnPropNearPlayer(testutil.InitTestWorld(t), "barrel"))
	})
}

func TestSpawnEnemyNearPlayer(t *testing.T) {
	t.Parallel()

	t.Run("近傍に敵を生成する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)

		target := consts.Coord[consts.Tile]{X: 13, Y: 5} // playerX+8
		before := countGridEntitiesAt(t, world, target)
		require.NoError(t, spawnEnemyNearPlayer(world, "moss_turtle"))
		assert.Equal(t, before+1, countGridEntitiesAt(t, world, target), "近傍に敵を1体生成する")
	})

	t.Run("プレイヤー不在はエラーを返す", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, spawnEnemyNearPlayer(testutil.InitTestWorld(t), "moss_turtle"))
	})
}

func TestSpawnStorageWithItems(t *testing.T) {
	t.Parallel()

	t.Run("収納propを生成する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)

		target := consts.Coord[consts.Tile]{X: 7, Y: 5} // playerX+2
		before := countGridEntitiesAt(t, world, target)
		require.NoError(t, spawnStorageWithItems(world))
		assert.Equal(t, before+1, countGridEntitiesAt(t, world, target), "収納 prop を1体生成する")
	})

	t.Run("プレイヤー不在はエラーを返す", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, spawnStorageWithItems(testutil.InitTestWorld(t)))
	})
}
