package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
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

func TestCanAddToStorage_ちょうど上限なら追加できる(t *testing.T) {
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
	world.Components.Weight.Add(item, &gc.Weight{Milligram: consts.MustParseWeight("1 kg")})

	assert.False(t, query.CanAddToStorage(world, storage, item))
}
