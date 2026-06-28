package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetWeapons は武器一覧を取得する（スロット1〜5）
// 必ず長さ5のスライスを返す
func GetWeapons(world w.World, owner ecs.Entity) []*ecs.Entity {
	weapons := make([]*ecs.Entity, 5)

	world.Manager.Join(
		world.Components.LocationEquipped,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		cat, _ := world.Components.CategoryOf(gc.InventoryCategoryKey, entity)
		if cat != gc.CategoryWeapon {
			return
		}
		equipped := world.Components.LocationEquipped.Get(entity).(*gc.LocationEquipped)
		if owner == equipped.Owner {
			if equipped.EquipmentSlot >= gc.SlotWeapon1 && equipped.EquipmentSlot <= gc.SlotWeapon5 {
				index := int(equipped.EquipmentSlot) - int(gc.SlotWeapon1)
				weapons[index] = &entity
			}
		}
	}))

	return weapons
}

// GetArmorEquipments は防具一覧を取得する（HEAD, TORSO, ARMS, HANDS, LEGS, FEET, JEWELRY）
// 必ず長さ7のスライスを返す
func GetArmorEquipments(world w.World, owner ecs.Entity) []*ecs.Entity {
	entities := make([]*ecs.Entity, 7)

	world.Manager.Join(
		world.Components.LocationEquipped,
		world.Components.Wearable,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		equipped := world.Components.LocationEquipped.Get(entity).(*gc.LocationEquipped)
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
				panic(fmt.Sprintf("不正な装備スロット: %v", equipped.EquipmentSlot))
			}
		}
	}))

	return entities
}
