package query

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// HeatContent はエンティティの燃焼熱量を Material と Weight から導く。燃料の行の物は残量の割合を掛ける。
// 値を保持せず読み取り時に算出する。どちらかの component を欠けば0
func HeatContent(world w.World, entity ecs.Entity) consts.Heat {
	if !world.Components.Material.Has(entity) || !world.Components.Weight.Has(entity) {
		return 0
	}
	kind := world.Components.Material.Get(entity).Kind
	weight := world.Components.Weight.Get(entity).Milligram
	heat := world.Resources.MaterialDefs.HeatOf(kind, weight)
	if _, fuel := FuelKindOf(world, entity); !fuel || !world.Components.Charge.Has(entity) || heat <= 0 {
		return heat
	}
	// 丸めで0になると燃やせる物から外れるので、残量があれば1は残す
	charge := world.Components.Charge.Get(entity)
	if charge.Current <= 0 {
		return 0
	}
	return max(1, heat*consts.Heat(charge.Current)/consts.Heat(charge.Max))
}

// IsCombustible は燃やせるか、すなわち燃焼熱量が正かを返す。可燃マーカーの代わりに使う
func IsCombustible(world w.World, entity ecs.Entity) bool {
	return HeatContent(world, entity) > 0
}
