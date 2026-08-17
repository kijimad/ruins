package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
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

// GetEntityWeight はエンティティ1個の重量を返す。1個1エンティティなので個数は掛けない。
// 総重量は所有エンティティを1個ずつ合算して求める。
func GetEntityWeight(world w.World, entity ecs.Entity) consts.Milligram {
	if !world.Components.Weight.Has(entity) {
		return 0
	}
	return world.Components.Weight.Get(entity).Milligram
}

// GetStorageCurrentWeight は収納の現在重量を返す
func GetStorageCurrentWeight(world w.World, storage ecs.Entity) consts.Milligram {
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

// CanAddStackToStorage は items をまとめて収納に追加できるか、合計重量で判定する。
// スタック丸ごとの移動可否を1回で判定するために使う。
func CanAddStackToStorage(world w.World, storage ecs.Entity, items []ecs.Entity) bool {
	if !world.Components.WeightCapacity.Has(storage) {
		return false
	}
	var total consts.Milligram
	for _, it := range items {
		total += GetEntityWeight(world, it)
	}
	wc := world.Components.WeightCapacity.Get(storage)
	return wc.Current+total <= wc.Max
}
