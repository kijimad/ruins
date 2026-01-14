package worldhelper

import (
	gc "github.com/kijimaD/ruins/lib/components"
	w "github.com/kijimaD/ruins/lib/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

const (
	// BaseCarryingWeight は基本所持可能重量(kg)
	BaseCarryingWeight = 10.0
	// StrengthWeightMultiplier は筋力1あたりの追加所持可能重量(kg)
	StrengthWeightMultiplier = 2.0
)

// CalculateMaxCarryingWeight は筋力ステータスから所持可能重量を計算する
// 計算式: 基本値 + (筋力 × 倍率)
func CalculateMaxCarryingWeight(attributes *gc.Attributes) float64 {
	if attributes == nil {
		return BaseCarryingWeight
	}
	strength := attributes.Strength.Base + attributes.Strength.Modifier
	return BaseCarryingWeight + float64(strength)*StrengthWeightMultiplier
}

// CalculateCurrentCarryingWeight は所持アイテムの総重量を計算する
// バックパック内と装備中のアイテムの重量を合算する
func CalculateCurrentCarryingWeight(world w.World, entity ecs.Entity) float64 {
	var totalWeight float64

	// 全アイテムを走査
	world.Manager.Join(
		world.Components.Item,
		world.Components.Weight,
	).Visit(ecs.Visit(func(itemEntity ecs.Entity) {
		weight := world.Components.Weight.Get(itemEntity).(*gc.Weight)

		// バックパック内のアイテム
		if itemEntity.HasComponent(world.Components.ItemLocationInBackpack) {
			count := 1
			// スタック可能アイテムの場合は個数を考慮
			if itemEntity.HasComponent(world.Components.Stackable) {
				stackable := world.Components.Stackable.Get(itemEntity).(*gc.Stackable)
				count = stackable.Count
			}
			totalWeight += weight.Kg * float64(count)
		}

		// 装備中のアイテム
		if itemEntity.HasComponent(world.Components.ItemLocationEquipped) {
			equipped := world.Components.ItemLocationEquipped.Get(itemEntity).(*gc.LocationEquipped)
			// このエンティティが装備しているアイテムのみ
			if equipped.Owner == entity {
				totalWeight += weight.Kg
			}
		}
	}))

	return totalWeight
}

// UpdateCarryingWeight はエンティティの所持重量プールを更新する
// 現在の所持重量と所持可能重量を再計算してPoolsコンポーネントに反映する
func UpdateCarryingWeight(world w.World, entity ecs.Entity) {
	if !entity.HasComponent(world.Components.Pools) {
		return
	}
	if !entity.HasComponent(world.Components.Attributes) {
		return
	}

	pools := world.Components.Pools.Get(entity).(*gc.Pools)
	attributes := world.Components.Attributes.Get(entity).(*gc.Attributes)

	// 所持可能重量を計算
	maxWeight := CalculateMaxCarryingWeight(attributes)
	pools.Weight.Max = maxWeight

	// 現在の所持重量を計算
	currentWeight := CalculateCurrentCarryingWeight(world, entity)
	pools.Weight.Current = currentWeight
}
