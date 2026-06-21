package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// MoveToBackpack はエンティティをバックパックに移動する
func MoveToBackpack(world w.World, entity ecs.Entity, owner ecs.Entity) {
	setLocation(world, entity, &gc.LocationInBackpack{Owner: owner})
	owner.AddComponent(world.Components.StatsChanged, &gc.StatsChanged{})
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// MoveToEquip はエンティティを指定スロットに装備する
func MoveToEquip(world w.World, entity ecs.Entity, owner ecs.Entity, slot gc.EquipmentSlotNumber) {
	setLocation(world, entity, &gc.LocationEquipped{
		Owner:         owner,
		EquipmentSlot: slot,
	})
	owner.AddComponent(world.Components.StatsChanged, &gc.StatsChanged{})
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// MoveToField はエンティティをフィールドに移動する
func MoveToField(world w.World, entity ecs.Entity, owner ecs.Entity) {
	setLocation(world, entity, &gc.LocationOnField{})
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// MoveToStorage はエンティティを収納に移動する
func MoveToStorage(world w.World, entity ecs.Entity, storage ecs.Entity) {
	setLocation(world, entity, &gc.LocationInStorage{Owner: storage})
}

// GetStorageItems は収納内のアイテムを取得する
func GetStorageItems(world w.World, storage ecs.Entity) []ecs.Entity {
	var items []ecs.Entity
	world.Manager.Join(world.Components.LocationInStorage).Visit(ecs.Visit(func(entity ecs.Entity) {
		loc := world.Components.LocationInStorage.Get(entity).(*gc.LocationInStorage)
		if loc.Owner == storage {
			items = append(items, entity)
		}
	}))
	return items
}

// GetStorageCurrentWeight は収納内アイテムの合計重量を返す
func GetStorageCurrentWeight(world w.World, storage ecs.Entity) float64 {
	var total float64
	for _, item := range GetStorageItems(world, storage) {
		if item.HasComponent(world.Components.Weight) {
			weightComp := world.Components.Weight.Get(item).(*gc.Weight)
			count := 1
			if item.HasComponent(world.Components.Stackable) {
				count = world.Components.Stackable.Get(item).(*gc.Stackable).Count
			}
			total += weightComp.Kg * float64(count)
		}
	}
	return total
}

// CanAddToStorage は収納にアイテムを追加できるか判定する
func CanAddToStorage(world w.World, storage ecs.Entity, item ecs.Entity) bool {
	if !storage.HasComponent(world.Components.Storage) {
		return false
	}
	storageComp := world.Components.Storage.Get(storage).(*gc.Storage)

	var itemWeight float64
	if item.HasComponent(world.Components.Weight) {
		weightComp := world.Components.Weight.Get(item).(*gc.Weight)
		count := 1
		if item.HasComponent(world.Components.Stackable) {
			count = world.Components.Stackable.Get(item).(*gc.Stackable).Count
		}
		itemWeight = weightComp.Kg * float64(count)
	}

	return GetStorageCurrentWeight(world, storage)+itemWeight <= storageComp.MaxWeight
}

// setLocation はエンティティの位置を設定する。排他制御を保証する。
// 既存の位置コンポーネントをすべて削除してから、新しい位置を設定する。
// 内部用関数なので直接呼び出さず、MoveToBackpack, MoveToField等を使用すること
func setLocation(world w.World, entity ecs.Entity, data interface{}) {
	// すべての位置コンポーネントを削除（排他制御）
	entity.RemoveComponent(world.Components.LocationInBackpack)
	entity.RemoveComponent(world.Components.LocationEquipped)
	entity.RemoveComponent(world.Components.LocationOnField)
	entity.RemoveComponent(world.Components.LocationInStorage)

	// dataの型に応じて位置コンポーネントを追加
	switch v := data.(type) {
	case *gc.LocationInBackpack:
		entity.AddComponent(world.Components.LocationInBackpack, v)
	case *gc.LocationEquipped:
		entity.AddComponent(world.Components.LocationEquipped, v)
	case *gc.LocationOnField:
		entity.AddComponent(world.Components.LocationOnField, v)
	case *gc.LocationInStorage:
		entity.AddComponent(world.Components.LocationInStorage, v)
	}
}
