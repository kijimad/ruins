package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestGetStorageItems_収納内のアイテムだけを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storage := world.ECS.NewEntity()
	otherStorage := world.ECS.NewEntity()

	item := world.ECS.NewEntity()
	world.Components.LocationInStorage.Add(item, &gc.LocationInStorage{Owner: storage})

	otherItem := world.ECS.NewEntity()
	world.Components.LocationInStorage.Add(otherItem, &gc.LocationInStorage{Owner: otherStorage})

	items := query.GetStorageItems(world, storage)
	assert.Equal(t, []ecs.Entity{item}, items)
}

func TestGetStorageItems_収納内が空なら空を返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storage := world.ECS.NewEntity()
	items := query.GetStorageItems(world, storage)
	assert.Empty(t, items)
}

func TestGetStorageCurrentWeight_WeightCapacityがあれば現在重量を返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storage := world.ECS.NewEntity()
	world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{
		Current: consts.MustParseWeight("3 kg"),
		Max:     consts.MustParseWeight("10 kg"),
	})

	got := query.GetStorageCurrentWeight(world, storage)
	assert.Equal(t, consts.MustParseWeight("3 kg"), got)
}

func TestGetStorageCurrentWeight_WeightCapacityが無ければ0(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storage := world.ECS.NewEntity()
	got := query.GetStorageCurrentWeight(world, storage)
	assert.Equal(t, consts.Milligram(0), got)
}

func TestCanAddToStorage_容量内なら追加できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storage := world.ECS.NewEntity()
	world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{
		Current: consts.MustParseWeight("3 kg"),
		Max:     consts.MustParseWeight("10 kg"),
	})

	item := world.ECS.NewEntity()
	world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("5 kg")})

	assert.True(t, query.CanAddToStorage(world, storage, item))
}

func TestCanAddToStorage_容量を超えるなら追加できない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storage := world.ECS.NewEntity()
	world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{
		Current: consts.MustParseWeight("8 kg"),
		Max:     consts.MustParseWeight("10 kg"),
	})

	item := world.ECS.NewEntity()
	world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("5 kg")})

	assert.False(t, query.CanAddToStorage(world, storage, item))
}

func TestCanAddToStorage_残量ちょうどなら追加できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storage := world.ECS.NewEntity()
	world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{
		Current: consts.MustParseWeight("5 kg"),
		Max:     consts.MustParseWeight("10 kg"),
	})

	item := world.ECS.NewEntity()
	world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("5 kg")})

	assert.True(t, query.CanAddToStorage(world, storage, item))
}

func TestCanAddToStorage_WeightCapacityが無ければ追加できない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storage := world.ECS.NewEntity()
	item := world.ECS.NewEntity()

	assert.False(t, query.CanAddToStorage(world, storage, item))
}

// TestCanAddStackToStorage は束の合計重量で追加可否を判定することを固定する。
// 個々には収まる品でも、束の合計が容量を超えるなら丸ごと拒否する
func TestCanAddStackToStorage(t *testing.T) {
	t.Parallel()

	newStorage := func(world w.World, current, capMax string) ecs.Entity {
		storage := world.ECS.NewEntity()
		world.Components.WeightCapacity.Add(storage, &gc.WeightCapacity{
			Current: consts.MustParseWeight(current),
			Max:     consts.MustParseWeight(capMax),
		})
		return storage
	}
	newItems := func(world w.World, each string, count int) []ecs.Entity {
		items := make([]ecs.Entity, count)
		for i := range count {
			e := world.ECS.NewEntity()
			world.Components.Weight.Add(e, &gc.Weight{Milligram: consts.MustParseWeight(each)})
			items[i] = e
		}
		return items
	}

	t.Run("合計が残量ちょうどなら追加できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		storage := newStorage(world, "4 kg", "10 kg")
		assert.True(t, query.CanAddStackToStorage(world, storage, newItems(world, "2 kg", 3)))
	})

	t.Run("合計が残量を超えるなら丸ごと拒否する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		storage := newStorage(world, "5 kg", "10 kg")
		// 1個2kgは個々には収まるが、3個の合計6kgは残量5kgを超える
		assert.False(t, query.CanAddStackToStorage(world, storage, newItems(world, "2 kg", 3)))
	})

	t.Run("WeightCapacityが無ければ追加できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		storage := world.ECS.NewEntity()
		assert.False(t, query.CanAddStackToStorage(world, storage, newItems(world, "1 kg", 1)))
	})
}
