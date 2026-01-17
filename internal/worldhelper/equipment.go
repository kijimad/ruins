package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// Equip は装備する
func Equip(world w.World, item ecs.Entity, owner ecs.Entity, slotNumber gc.EquipmentSlotNumber) {
	item.AddComponent(world.Components.ItemLocationEquipped, &gc.LocationEquipped{Owner: owner, EquipmentSlot: slotNumber})
	item.RemoveComponent(world.Components.ItemLocationInPlayerBackpack)
	item.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
}

// Disarm は装備を外す
func Disarm(world w.World, item ecs.Entity) {
	item.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})
	item.RemoveComponent(world.Components.ItemLocationEquipped)
	item.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
}

// GetMeleeWeapon は近接武器を取得する
func GetMeleeWeapon(world w.World, owner ecs.Entity) *ecs.Entity {
	var result *ecs.Entity

	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationEquipped,
		world.Components.Weapon,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		equipped := world.Components.ItemLocationEquipped.Get(entity).(*gc.LocationEquipped)
		if owner == equipped.Owner && equipped.EquipmentSlot == gc.SlotMeleeWeapon {
			result = &entity
		}
	}))

	return result
}

// GetRangedWeapon は遠距離武器を取得する
func GetRangedWeapon(world w.World, owner ecs.Entity) *ecs.Entity {
	var result *ecs.Entity

	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationEquipped,
		world.Components.Weapon,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		equipped := world.Components.ItemLocationEquipped.Get(entity).(*gc.LocationEquipped)
		if owner == equipped.Owner && equipped.EquipmentSlot == gc.SlotRangedWeapon {
			result = &entity
		}
	}))

	return result
}

// GetArmorEquipments は防具一覧を取得する（HEAD, TORSO, LEGS, JEWELRY）
// 必ず長さ4のスライスを返す
func GetArmorEquipments(world w.World, owner ecs.Entity) []*ecs.Entity {
	entities := make([]*ecs.Entity, 4)

	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationEquipped,
		world.Components.Wearable,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		equipped := world.Components.ItemLocationEquipped.Get(entity).(*gc.LocationEquipped)
		if owner == equipped.Owner {
			// スロット番号から配列インデックスを決定
			switch equipped.EquipmentSlot {
			case gc.SlotHead:
				entities[0] = &entity
			case gc.SlotTorso:
				entities[1] = &entity
			case gc.SlotLegs:
				entities[2] = &entity
			case gc.SlotJewelry:
				entities[3] = &entity
			}
		}
	}))

	return entities
}
