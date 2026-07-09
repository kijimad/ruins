package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetStorageItems は収納内のアイテムを取得する
func GetStorageItems(world w.World, storage ecs.Entity) []ecs.Entity {
	var items []ecs.Entity
	world.Manager.Join(world.Components.LocationInStorage).Visit(ecs.Visit(func(entity ecs.Entity) {
		loc := world.Components.LocationInStorage.MustGet(entity)
		if loc.Owner == storage {
			items = append(items, entity)
		}
	}))
	return items
}

// GetEntityWeight はエンティティの総重量を返す。Stackableの場合は個数を掛ける
func GetEntityWeight(world w.World, entity ecs.Entity) float64 {
	if !entity.HasComponent(world.Components.Weight) {
		return 0
	}
	weightComp := world.Components.Weight.MustGet(entity)
	count := GetEntityCount(world, entity)
	return weightComp.Kg * float64(count)
}

// GetStorageCurrentWeight は収納の現在重量を返す
func GetStorageCurrentWeight(world w.World, storage ecs.Entity) float64 {
	if !storage.HasComponent(world.Components.WeightCapacity) {
		return 0
	}
	wc := world.Components.WeightCapacity.MustGet(storage)
	return wc.Current
}

// CanAddToStorage は収納にアイテムを追加できるか判定する
func CanAddToStorage(world w.World, storage ecs.Entity, item ecs.Entity) bool {
	if !storage.HasComponent(world.Components.WeightCapacity) {
		return false
	}
	wc := world.Components.WeightCapacity.MustGet(storage)
	return wc.Current+GetEntityWeight(world, item) <= wc.Max
}
