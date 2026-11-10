package query

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// HeatContent はエンティティの燃焼熱量を Material と Weight から導く。
// 値を保持せず読み取り時に算出する。どちらかの component を欠けば0
func HeatContent(world w.World, entity ecs.Entity) consts.Heat {
	if !world.Components.Material.Has(entity) || !world.Components.Weight.Has(entity) {
		return 0
	}
	kind := world.Components.Material.Get(entity).Kind
	weight := world.Components.Weight.Get(entity).Milligram
	return world.Resources.MaterialDefs.HeatOf(kind, weight)
}

// IsCombustible は燃やせるか、すなわち燃焼熱量が正かを返す。可燃マーカーの代わりに使う
func IsCombustible(world w.World, entity ecs.Entity) bool {
	return HeatContent(world, entity) > 0
}
