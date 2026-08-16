package query

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 価格倍率
const (
	BuyPriceMultiplier  = 2.0 // 購入価格は価値の2倍
	SellPriceMultiplier = 0.5 // 売却価格は価値の半分
)

// CalculateBuyPrice は購入価格を計算する（価値の2倍）
func CalculateBuyPrice(baseValue int) consts.Currency {
	return consts.Currency(float64(baseValue) * BuyPriceMultiplier)
}

// CalculateSellPrice は売却価格を計算する（価値の半分）
func CalculateSellPrice(baseValue int) consts.Currency {
	return consts.Currency(float64(baseValue) * SellPriceMultiplier)
}

// BuyPrice はプレイヤーが entity を買うときの、交渉スキルの買値倍率込みの購入価格を返す。
// base は価値と個数の積。店頭の表示価格と取引価格をこの1関数で揃える
func BuyPrice(world w.World, player ecs.Entity, entity ecs.Entity) consts.Currency {
	base := GetItemValue(world, entity) * GetEntityCount(world, entity)
	price := CalculateBuyPrice(base)
	if world.Components.CharModifiers.Has(player) {
		price = consts.Currency(world.Components.CharModifiers.Get(player).BuyPrice.ApplyInt(int(price)))
	}
	return price
}

// SellPrice はプレイヤーが entity を売るときの、交渉スキルの売値倍率込みの売却価格を返す。
// base は価値と個数の積。店頭の表示価格と取引価格をこの1関数で揃える。
// 無価値な品は素直に 0 を返す。売却自体は他の品と同様に可能で、対価が 0 になるだけ
func SellPrice(world w.World, player ecs.Entity, entity ecs.Entity) consts.Currency {
	base := GetItemValue(world, entity) * GetEntityCount(world, entity)
	price := CalculateSellPrice(base)
	if world.Components.CharModifiers.Has(player) {
		price = consts.Currency(world.Components.CharModifiers.Get(player).SellPrice.ApplyInt(int(price)))
	}
	return price
}

// GetItemValue はアイテムの基本価値を取得する
func GetItemValue(world w.World, entity ecs.Entity) int {
	if !world.Components.Value.Has(entity) {
		return 0
	}
	return world.Components.Value.Get(entity).Value
}
