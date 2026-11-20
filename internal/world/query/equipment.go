package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// IsHoldable はエンティティを手持ちの欄に持てるかを返す。近接または射撃のいずれかを持てば持てる。
func IsHoldable(world w.World, entity ecs.Entity) bool {
	return world.Components.Melee.Has(entity) || world.Components.Fire.Has(entity)
}

// GetHeld は手持ちの欄ごとの物を、SlotHand1 から順に並べて返す。空きスロットは nil
func GetHeld(world w.World, owner ecs.Entity) [gc.HandSlotCount]*ecs.Entity {
	var weapons [gc.HandSlotCount]*ecs.Entity

	weaponsQuery := ecs.NewFilter1[gc.LocationEquipped](world.ECS).Query()
	for weaponsQuery.Next() {
		entity := weaponsQuery.Entity()
		if !IsHoldable(world, entity) {
			continue
		}
		equipped := world.Components.LocationEquipped.Get(entity)
		if owner == equipped.Owner {
			if i, ok := equipped.EquipmentSlot.HandIndex(); ok {
				weapons[i] = &entity
			}
		}
	}

	return weapons
}

// SelectedWeapon は owner が選択中の手持ちの欄に装備している武器を返す。空きスロットなら ok は偽。
// 選択が手持ちの欄であることはロードの検査と数字キーの処理が保証するので、破れていれば panic する
func SelectedWeapon(world w.World, owner ecs.Entity) (ecs.Entity, bool) {
	slot := GetWeaponSelection(world).Slot
	i, ok := slot.HandIndex()
	if !ok {
		panic(fmt.Sprintf("weapon selection is not a weapon slot: %d", slot))
	}
	weapon := GetHeld(world, owner)[i]
	if weapon == nil {
		return ecs.Entity{}, false
	}
	return *weapon, true
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
