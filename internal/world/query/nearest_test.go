package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestFindNearestEntity(t *testing.T) {
	t.Parallel()

	t.Run("最寄りのエンティティを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		self := world.Manager.NewEntity()
		self.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		near := world.Manager.NewEntity()
		near.AddComponent(world.Components.GridElement, &gc.GridElement{X: 6, Y: 5})

		far := world.Manager.NewEntity()
		far.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

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

		self := world.Manager.NewEntity()
		self.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		world.Manager.NewEntity().AddComponent(world.Components.GridElement, &gc.GridElement{X: 8, Y: 5})

		closest := world.Manager.NewEntity()
		closest.AddComponent(world.Components.GridElement, &gc.GridElement{X: 6, Y: 6})

		world.Manager.NewEntity().AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

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

		self := world.Manager.NewEntity()
		self.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		from := &gc.GridElement{X: 5, Y: 5}
		found, _, _ := query.FindNearestEntity(world, self, from, func(_ ecs.Entity) bool {
			return true
		})

		assert.Nil(t, found)
	})

	t.Run("Deadエンティティは除外される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		self := world.Manager.NewEntity()
		self.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		dead := world.Manager.NewEntity()
		dead.AddComponent(world.Components.GridElement, &gc.GridElement{X: 6, Y: 5})
		dead.AddComponent(world.Components.Dead, &gc.Dead{})

		from := &gc.GridElement{X: 5, Y: 5}
		found, _, _ := query.FindNearestEntity(world, self, from, func(_ ecs.Entity) bool {
			return true
		})

		assert.Nil(t, found)
	})

	t.Run("条件に一致しない場合はnilを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		self := world.Manager.NewEntity()
		self.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		world.Manager.NewEntity().AddComponent(world.Components.GridElement, &gc.GridElement{X: 6, Y: 5})

		from := &gc.GridElement{X: 5, Y: 5}
		found, grid, dist := query.FindNearestEntity(world, self, from, func(_ ecs.Entity) bool {
			return false
		})

		assert.Nil(t, found)
		assert.Nil(t, grid)
		assert.Equal(t, -1, dist)
	})
}
