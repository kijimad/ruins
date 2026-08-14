package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// IsWeapon はエンティティが武器かを返す。近接または射撃のいずれかを持てば武器とみなす。
func IsWeapon(world w.World, entity ecs.Entity) bool {
	return world.Components.Melee.Has(entity) || world.Components.Fire.Has(entity)
}

// GetWeapons は武器一覧を取得する（スロット1〜5）
// 必ず長さ5のスライスを返す
func GetWeapons(world w.World, owner ecs.Entity) []*ecs.Entity {
	weapons := make([]*ecs.Entity, 5)

	weaponsQuery := ecs.NewFilter1[gc.LocationEquipped](world.ECS).Query()
	for weaponsQuery.Next() {
		entity := weaponsQuery.Entity()
		if !IsWeapon(world, entity) {
			continue
		}
		equipped := world.Components.LocationEquipped.Get(entity)
		if owner == equipped.Owner {
			if equipped.EquipmentSlot >= gc.SlotWeapon1 && equipped.EquipmentSlot <= gc.SlotWeapon5 {
				index := int(equipped.EquipmentSlot) - int(gc.SlotWeapon1)
				weapons[index] = &entity
			}
		}
	}

	return weapons
}

// GetArmorEquipments は防具一覧を取得する（HEAD, TORSO, ARMS, HANDS, LEGS, FEET, JEWELRY）
// 必ず長さ7のスライスを返す
func GetArmorEquipments(world w.World, owner ecs.Entity) []*ecs.Entity {
	entities := make([]*ecs.Entity, 7)

	armorQuery := ecs.NewFilter2[gc.LocationEquipped, gc.Wearable](world.ECS).Query()
	for armorQuery.Next() {
		entity := armorQuery.Entity()
		equipped := world.Components.LocationEquipped.Get(entity)
		if owner == equipped.Owner {
			switch equipped.EquipmentSlot {
			case gc.SlotHead:
				entities[0] = &entity
			case gc.SlotTorso:
				entities[1] = &entity
			case gc.SlotArms:
				entities[2] = &entity
			case gc.SlotHands:
				entities[3] = &entity
			case gc.SlotLegs:
				entities[4] = &entity
			case gc.SlotFeet:
				entities[5] = &entity
			case gc.SlotJewelry:
				entities[6] = &entity
			default:
				panic(fmt.Sprintf("invalid equipment slot: %v", equipped.EquipmentSlot))
			}
		}
	}

	return entities
}
