package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindNearestEntity(t *testing.T) {
	t.Parallel()

	t.Run("最寄りのエンティティを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		self := world.ECS.NewEntity()
		world.Components.GridElement.Add(self, &gc.GridElement{X: 5, Y: 5})

		near := world.ECS.NewEntity()
		world.Components.GridElement.Add(near, &gc.GridElement{X: 6, Y: 5})

		far := world.ECS.NewEntity()
		world.Components.GridElement.Add(far, &gc.GridElement{X: 10, Y: 10})

		from := &gc.GridElement{X: 5, Y: 5}
		found, grid, dist := query.FindNearestEntity(world, self, from, func(_ ecs.Entity) bool {
			return true
		})

		assert.NotNil(t, found)
		assert.Equal(t, consts.Tile(6), grid.X)
		assert.Equal(t, consts.Tile(5), grid.Y)
		assert.Equal(t, 1, dist)
	})

	t.Run("複数候補から最も近いものを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		self := world.ECS.NewEntity()
		world.Components.GridElement.Add(self, &gc.GridElement{X: 5, Y: 5})

		world.Components.GridElement.NewEntity(&gc.GridElement{X: 8, Y: 5})

		closest := world.ECS.NewEntity()
		world.Components.GridElement.Add(closest, &gc.GridElement{X: 6, Y: 6})

		world.Components.GridElement.NewEntity(&gc.GridElement{X: 10, Y: 10})

		from := &gc.GridElement{X: 5, Y: 5}
		found, grid, dist := query.FindNearestEntity(world, self, from, func(_ ecs.Entity) bool {
			return true
		})

		assert.NotNil(t, found)
		assert.Equal(t, consts.Tile(6), grid.X)
		assert.Equal(t, consts.Tile(6), grid.Y)
		assert.Equal(t, 1, dist)
	})

	t.Run("selfは除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		self := world.ECS.NewEntity()
		world.Components.GridElement.Add(self, &gc.GridElement{X: 5, Y: 5})

		from := &gc.GridElement{X: 5, Y: 5}
		found, _, _ := query.FindNearestEntity(world, self, from, func(_ ecs.Entity) bool {
			return true
		})

		assert.Nil(t, found)
	})

	t.Run("Deadエンティティは除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		self := world.ECS.NewEntity()
		world.Components.GridElement.Add(self, &gc.GridElement{X: 5, Y: 5})

		dead := world.ECS.NewEntity()
		world.Components.GridElement.Add(dead, &gc.GridElement{X: 6, Y: 5})
		world.Components.Dead.Add(dead, &gc.Dead{})

		from := &gc.GridElement{X: 5, Y: 5}
		found, _, _ := query.FindNearestEntity(world, self, from, func(_ ecs.Entity) bool {
			return true
		})

		assert.Nil(t, found)
	})

	t.Run("条件に一致しない場合はnilを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		self := world.ECS.NewEntity()
		world.Components.GridElement.Add(self, &gc.GridElement{X: 5, Y: 5})

		world.Components.GridElement.NewEntity(&gc.GridElement{X: 6, Y: 5})

		from := &gc.GridElement{X: 5, Y: 5}
		found, grid, dist := query.FindNearestEntity(world, self, from, func(_ ecs.Entity) bool {
			return false
		})

		assert.Nil(t, found)
		assert.Nil(t, grid)
		assert.Equal(t, -1, dist)
	})
}

// TestFindNearestCharacter_タイルを無視する は B2 最適化のガード。
// 床/壁を模した非キャラの GridElement エンティティが多数あっても、キャラクターだけを対象に
// 最寄りを返すことを固定する（インデックス経由でタイルを走査しない契約）。
func TestFindNearestCharacter_タイルを無視する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := world.Components.Dungeon.Get(world.Resources.SingletonEntity)
	d.Level = gc.Level{TileWidth: consts.Tile(50), TileHeight: consts.Tile(50)}
	player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// self となる敵。プレイヤーが最寄りのキャラクター（距離3）
	enemy, err := lifecycle.SpawnEnemy(world, 13, 10, "火の玉")
	require.NoError(t, err)

	// 敵のすぐ隣(距離1〜2)に非キャラの GridElement エンティティ（タイル模擬）を多数置く。
	// 全走査ならこれらが最寄り候補になり得るが、キャラ探索では無視されるべき
	for i := range 50 {
		world.Components.GridElement.NewEntity(&gc.GridElement{X: consts.Tile(12), Y: consts.Tile(9 + i%3)})
	}

	enemyGrid := world.Components.GridElement.Get(enemy)
	found, _, dist := query.FindNearestCharacter(world, enemy, enemyGrid, func(e ecs.Entity) bool {
		return world.Components.Player.Has(e)
	})

	require.NotNil(t, found, "キャラクター（プレイヤー）が見つかる")
	assert.Equal(t, player, *found, "非キャラのタイルは無視し、プレイヤーを返す")
	assert.Equal(t, 3, dist, "距離はプレイヤーまでの3（隣接タイルではない）")
}
