package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
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
	if !world.Components.Value.Has(entity) {
		return 0
	}
	return world.Components.Value.Get(entity).Value
}

// recruitValueMultiplier は隊員候補の能力値合計に掛ける基準価値の係数
const recruitValueMultiplier = 30

// IsRecruit は商人の収納内の実体が隊員候補かを返す。収納にはアイテムと隊員候補しか入らないため、
// アイテムの類型に当てはまらない実体を隊員候補とみなす。エンティティの類型判定は
// Components.CategoryOf に集約されており、コンポーネント構成から求める。収納外の実体には使わない
func IsRecruit(world w.World, entity ecs.Entity) bool {
	_, isItem := world.Components.CategoryOf(gc.InventoryCategoryKey, entity)
	return !isItem
}

// RecruitValue は隊員候補の基準価値を能力値合計から算出する。売買価格の元になる
func RecruitValue(a gc.Abilities) int {
	total := a.Vitality.Base + a.Strength.Base + a.Sensation.Base +
		a.Dexterity.Base + a.Agility.Base + a.Defense.Base
	return total * recruitValueMultiplier
}

// StockBaseValue は在庫実体1件の基準価値を返す。隊員候補は能力値から、アイテムは Value から出し、
// 個数を掛けた実体まるごとの価値にする。買値・売値はこの値を CalculateBuyPrice/SellPrice に通して出す
func StockBaseValue(world w.World, entity ecs.Entity) int {
	count := GetEntityCount(world, entity)
	// IsRecruit はアイテムの類型でない実体を候補とみなすため、アイテムでも候補でもない裸の実体も
	// true になりうる。Abilities.Has は冗長でなく安全網で、能力を持たない実体を Value 経由へ落として
	// Abilities.Get の panic を防ぐ。実際の候補は SpawnStorageRecruit で必ず Abilities を持つ
	if IsRecruit(world, entity) && world.Components.Abilities.Has(entity) {
		return RecruitValue(*world.Components.Abilities.Get(entity)) * count
	}
	return GetItemValue(world, entity) * count
}
