package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// GetStorageItems は収納内のアイテムを取得する
func GetStorageItems(world w.World, storage ecs.Entity) []ecs.Entity {
	var items []ecs.Entity
	itemsQuery := ecs.NewFilter1[gc.LocationInStorage](world.ECS).Query()
	for itemsQuery.Next() {
		entity := itemsQuery.Entity()
		loc := world.Components.LocationInStorage.Get(entity)
		if loc.Owner == storage {
			items = append(items, entity)
		}
	}
	return items
}

// GetEntityWeight はエンティティの総重量を返す。Stackableの場合は個数を掛ける
func GetEntityWeight(world w.World, entity ecs.Entity) float64 {
	if !world.Components.Weight.Has(entity) {
		return 0
	}
	weightComp := world.Components.Weight.Get(entity)
	count := GetEntityCount(world, entity)
	return weightComp.Kg * float64(count)
}

// GetStorageCurrentWeight は収納の現在重量を返す
func GetStorageCurrentWeight(world w.World, storage ecs.Entity) float64 {
	if !world.Components.WeightCapacity.Has(storage) {
		return 0
	}
	return world.Components.WeightCapacity.Get(storage).Current
}

// CanAddToStorage は収納にアイテムを追加できるか判定する
func CanAddToStorage(world w.World, storage ecs.Entity, item ecs.Entity) bool {
	if !world.Components.WeightCapacity.Has(storage) {
		return false
	}
	wc := world.Components.WeightCapacity.Get(storage)
	return wc.Current+GetEntityWeight(world, item) <= wc.Max
}
