package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

const (
	// baseCarryingWeight は基本所持可能重量
	baseCarryingWeight = consts.Milligram(10 * consts.MilligramPerKg)
	// strengthWeightMultiplier は筋力1あたりの追加所持可能重量
	strengthWeightMultiplier = consts.Milligram(2 * consts.MilligramPerKg)
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
			maxWeight = consts.Milligram(mods.MaxWeight.ApplyInt(int(maxWeight)))
		}

		wc.Max = maxWeight
	}

	// Currentを再計算する
	wc.Current = calculateOwnedWeight(world, entity)
}

// calculateMaxCarryingWeight は筋力ステータスから所持可能重量を計算する
func calculateMaxCarryingWeight(abilities *gc.Abilities) consts.Milligram {
	if abilities == nil {
		return baseCarryingWeight
	}
	strength := abilities.Strength.Base + abilities.Strength.Modifier
	// strength は無次元の整数なので、Milligram との掛け算は次元を変えず mg のまま
	return baseCarryingWeight + consts.Milligram(strength)*strengthWeightMultiplier
}

// calculateOwnedWeight はエンティティが所有するアイテムの総重量を計算する。
// Backpack内、装備中、Storage内のアイテムをOwnerで判定して合算する
func calculateOwnedWeight(world w.World, entity ecs.Entity) consts.Milligram {
	var totalWeight consts.Milligram

	weightQuery := ecs.NewFilter1[gc.Weight](world.ECS).Query()
	for weightQuery.Next() {
		itemEntity := weightQuery.Entity()
		// アイテムは排他的に1箇所にのみ存在する（lifecycle の MoveToX が保証）ため else if で判定する
		switch {
		case world.Components.LocationInBackpack.Has(itemEntity):
			if world.Components.LocationInBackpack.Get(itemEntity).Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		case world.Components.LocationEquipped.Has(itemEntity):
			if world.Components.LocationEquipped.Get(itemEntity).Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		case world.Components.LocationInStorage.Has(itemEntity):
			if world.Components.LocationInStorage.Get(itemEntity).Owner == entity {
				totalWeight += GetEntityWeight(world, itemEntity)
			}
		}
	}

	return totalWeight
}
