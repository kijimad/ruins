package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// LocationType はエンティティの位置を表すenum型
type LocationType int

const (
	// LocationInBackpack はプレイヤーのバックパック内
	LocationInBackpack LocationType = iota
	// LocationEquipped は装備中
	LocationEquipped
	// LocationOnField はフィールド上
	LocationOnField
	// LocationInStorage は収納内
	LocationInStorage
)

// MoveToBackpack はエンティティをバックパックに移動する
func MoveToBackpack(world w.World, entity ecs.Entity, owner ecs.Entity) {
	backpackData := &gc.LocationInBackpack{Owner: owner}
	setLocation(world, entity, LocationInBackpack, backpackData)
	owner.AddComponent(world.Components.StatsChanged, &gc.StatsChanged{})
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// MoveToEquip はエンティティを指定スロットに装備する
func MoveToEquip(world w.World, entity ecs.Entity, owner ecs.Entity, slot gc.EquipmentSlotNumber) {
	equipData := &gc.LocationEquipped{
		Owner:         owner,
		EquipmentSlot: slot,
	}
	setLocation(world, entity, LocationEquipped, equipData)
	owner.AddComponent(world.Components.StatsChanged, &gc.StatsChanged{})
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// MoveToField はエンティティをフィールドに移動する
func MoveToField(world w.World, entity ecs.Entity, owner ecs.Entity) {
	setLocation(world, entity, LocationOnField, nil)
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// MoveToStorage はエンティティを収納に移動する
func MoveToStorage(world w.World, entity ecs.Entity, storage ecs.Entity) {
	storageData := &gc.LocationInStorage{Owner: storage}
	setLocation(world, entity, LocationInStorage, storageData)
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
			w := world.Components.Weight.Get(item).(*gc.Weight)
			count := 1
			if item.HasComponent(world.Components.Stackable) {
				count = world.Components.Stackable.Get(item).(*gc.Stackable).Count
			}
			total += w.Kg * float64(count)
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
		w := world.Components.Weight.Get(item).(*gc.Weight)
		count := 1
		if item.HasComponent(world.Components.Stackable) {
			count = world.Components.Stackable.Get(item).(*gc.Stackable).Count
		}
		itemWeight = w.Kg * float64(count)
	}

	return GetStorageCurrentWeight(world, storage)+itemWeight <= storageComp.MaxWeight
}

// setLocation はエンティティの位置を設定する。排他制御を保証する。
// 既存の位置コンポーネントをすべて削除してから、新しい位置を設定する。
// 内部用関数なので直接呼び出さず、MoveToBackpack, MoveToField等を使用すること
func setLocation(world w.World, entity ecs.Entity, locType LocationType, data interface{}) {
	// すべての位置コンポーネントを削除（排他制御）
	entity.RemoveComponent(world.Components.LocationInBackpack)
	entity.RemoveComponent(world.Components.LocationEquipped)
	entity.RemoveComponent(world.Components.LocationOnField)
	entity.RemoveComponent(world.Components.LocationInStorage)

	// 指定された位置コンポーネントを追加
	switch locType {
	case LocationInBackpack:
		entity.AddComponent(world.Components.LocationInBackpack, data.(*gc.LocationInBackpack))

	case LocationEquipped:
		entity.AddComponent(world.Components.LocationEquipped, data.(*gc.LocationEquipped))

	case LocationOnField:
		entity.AddComponent(world.Components.LocationOnField, &gc.LocationOnField{})

	case LocationInStorage:
		entity.AddComponent(world.Components.LocationInStorage, data.(*gc.LocationInStorage))
	}
}
