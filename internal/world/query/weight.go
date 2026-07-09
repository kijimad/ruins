package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

const (
	// baseCarryingWeight は基本所持可能重量(kg)
	baseCarryingWeight = 10.0
	// strengthWeightMultiplier は筋力1あたりの追加所持可能重量(kg)
	strengthWeightMultiplier = 2.0
)

// UpdateWeightCapacity はエンティティのWeightCapacityを更新する。
// PlayerはAbilitiesからMaxを計算し、Backpack+Equippedの重量をCurrentに設定する。
// StorageはMaxを変更せず、LocationInStorageの重量をCurrentに設定する
func UpdateWeightCapacity(world w.World, entity ecs.Entity) {
	if !entity.HasComponent(world.Components.WeightCapacity) {
		return
	}

	wc := world.Components.WeightCapacity.MustGet(entity)

	// Abilitiesを持つエンティティはMaxを再計算する（Player用）
	if entity.HasComponent(world.Components.Abilities) {
		abilities := world.Components.Abilities.MustGet(entity)
		maxWeight := calculateMaxCarryingWeight(abilities)

		if entity.HasComponent(world.Components.CharModifiers) {
			mods := world.Components.CharModifiers.MustGet(entity)
			maxWeight = maxWeight * float64(mods.MaxWeight) / 100
		}

		wc.Max = maxWeight
	}

	// Currentを再計算する
	wc.Current = calculateOwnedWeight(world, entity)
}

// calculateMaxCarryingWeight は筋力ステータスから所持可能重量を計算する
func calculateMaxCarryingWeight(abilities *gc.Abilities) float64 {
	if abilities == nil {
		return baseCarryingWeight
	}
	strength := abilities.Strength.Base + abilities.Strength.Modifier
	return baseCarryingWeight + float64(strength)*strengthWeightMultiplier
}

// calculateOwnedWeight はエンティティが所有するアイテムの総重量を計算する。
// Backpack内、装備中、Storage内のアイテムをOwnerで判定して合算する
func calculateOwnedWeight(world w.World, entity ecs.Entity) float64 {
	var totalWeight float64

	world.Manager.Join(
		world.Components.Weight,
	).Visit(ecs.Visit(func(itemEntity ecs.Entity) {
		if itemEntity.HasComponent(world.Components.LocationInBackpack) {
			loc := world.Components.LocationInBackpack.MustGet(itemEntity)
			if loc.Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		}

		if itemEntity.HasComponent(world.Components.LocationEquipped) {
			loc := world.Components.LocationEquipped.MustGet(itemEntity)
			if loc.Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		}

		if itemEntity.HasComponent(world.Components.LocationInStorage) {
			loc := world.Components.LocationInStorage.MustGet(itemEntity)
			if loc.Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		}
	}))

	return totalWeight
}
