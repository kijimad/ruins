package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// 価格倍率
const (
	BuyPriceMultiplier  = 2.0 // 購入価格は価値の2倍
	SellPriceMultiplier = 0.5 // 売却価格は価値の半分
)

// CalculateBuyPrice は購入価格を計算する（価値の2倍）
func CalculateBuyPrice(baseValue int) int {
	return int(float64(baseValue) * BuyPriceMultiplier)
}

// CalculateSellPrice は売却価格を計算する（価値の半分）
func CalculateSellPrice(baseValue int) int {
	return int(float64(baseValue) * SellPriceMultiplier)
}

// GetItemValue はアイテムの基本価値を取得する
func GetItemValue(world w.World, entity ecs.Entity) int {
	if !entity.HasComponent(world.Components.Value) {
		return 0
	}
	value := world.Components.Value.Get(entity).(*gc.Value)
	return value.Value
}
