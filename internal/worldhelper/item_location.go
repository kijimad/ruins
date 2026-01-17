package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ItemLocationType はアイテムの位置を表すenum型
type ItemLocationType int

const (
	// ItemLocationInPlayerBackpack はプレイヤーのバックパック内
	ItemLocationInPlayerBackpack ItemLocationType = iota
	// ItemLocationEquipped は装備中
	ItemLocationEquipped
	// ItemLocationOnField はフィールド上
	ItemLocationOnField
)

// MoveToBackpack はアイテムをバックパックに移動する
func MoveToBackpack(world w.World, item ecs.Entity, owner ecs.Entity) {
	setItemLocation(world, item, ItemLocationInPlayerBackpack, nil)
	owner.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// MoveToEquip はアイテムを指定スロットに装備する
func MoveToEquip(world w.World, item ecs.Entity, owner ecs.Entity, slot gc.EquipmentSlotNumber) {
	equipData := &gc.LocationEquipped{
		Owner:         owner,
		EquipmentSlot: slot,
	}
	setItemLocation(world, item, ItemLocationEquipped, equipData)
	owner.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// MoveToField はアイテムをフィールドに移動する
func MoveToField(world w.World, item ecs.Entity, owner ecs.Entity) {
	setItemLocation(world, item, ItemLocationOnField, nil)
	owner.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
}

// setItemLocation はアイテムの位置を設定（排他制御を保証）
// 既存の位置コンポーネントをすべて削除してから、新しい位置を設定する
// data は ItemLocationEquipped の場合に *gc.LocationEquipped を渡す
// 内部用関数なので直接呼び出さず、MoveToBackpack, DropToField等を使用すること
func setItemLocation(world w.World, item ecs.Entity, locType ItemLocationType, data interface{}) {
	// すべての位置コンポーネントを削除（排他制御）
	item.RemoveComponent(world.Components.ItemLocationInPlayerBackpack)
	item.RemoveComponent(world.Components.ItemLocationEquipped)
	item.RemoveComponent(world.Components.ItemLocationOnField)

	// 指定された位置コンポーネントを追加
	switch locType {
	case ItemLocationInPlayerBackpack:
		item.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})

	case ItemLocationEquipped:
		equipData := data.(*gc.LocationEquipped)
		item.AddComponent(world.Components.ItemLocationEquipped, equipData)

	case ItemLocationOnField:
		item.AddComponent(world.Components.ItemLocationOnField, &gc.LocationOnField{})
	}
}
