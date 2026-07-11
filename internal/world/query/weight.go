package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
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
	if !world.Components.WeightCapacity.Has(entity) {
		return
	}

	wc := world.Components.WeightCapacity.Get(entity)

	// Abilitiesを持つエンティティはMaxを再計算する（Player用）
	if world.Components.Abilities.Has(entity) {
		abilities := world.Components.Abilities.Get(entity)
		maxWeight := calculateMaxCarryingWeight(abilities)

		if world.Components.CharModifiers.Has(entity) {
			mods := world.Components.CharModifiers.Get(entity)
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

	weightQuery := ecs.NewFilter1[gc.Weight](world.ECS).Query()
	for weightQuery.Next() {
		itemEntity := weightQuery.Entity()
		if world.Components.LocationInBackpack.Has(itemEntity) {
			loc := world.Components.LocationInBackpack.Get(itemEntity)
			if loc.Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		}

		if world.Components.LocationEquipped.Has(itemEntity) {
			loc := world.Components.LocationEquipped.Get(itemEntity)
			if loc.Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		}

		if world.Components.LocationInStorage.Has(itemEntity) {
			loc := world.Components.LocationInStorage.Get(itemEntity)
			if loc.Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		}
	}

	return totalWeight
}
