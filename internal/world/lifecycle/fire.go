package lifecycle

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// LoadNextFuel は火の収納の最上段の燃料を燃やし始める。
// Burning.Remaining をその燃料の HeatContent×効率/100 にし、燃料を収納から取り除く。
// 燃料が無ければ false を返す。呼び出し側は fire に Burning を付けてから呼ぶ。
// 毎ターンの燃焼進行と着火の両方がこの一様な過程を共有する。効率は query.BurnEfficiency が場所から引く。
func LoadNextFuel(world w.World, fire ecs.Entity) bool {
	next, ok := firstFuelInStorage(world, fire)
	if !ok {
		return false
	}
	eff := query.BurnEfficiency(world, fire)
	burning := world.Components.Burning.Get(fire)
	burning.Remaining = world.Components.Fuel.Get(next).HeatContent * eff / 100
	consumeFuel(world, fire, next)
	return true
}

// firstFuelInStorage は火の収納の最上段の燃料を返す。収納の並びは決定的にソートし上から順に燃やす
func firstFuelInStorage(world w.World, fire ecs.Entity) (ecs.Entity, bool) {
	items := query.SortEntities(world, query.GetStorageItems(world, fire))
	for _, item := range items {
		if world.Components.Fuel.Has(item) {
			return item, true
		}
	}
	return ecs.Entity{}, false
}

// consumeFuel は燃やし始めた燃料を火の収納から取り除く。火が重量を持つなら再計算を促す
func consumeFuel(world w.World, fire ecs.Entity, fuel ecs.Entity) {
	world.ECS.RemoveEntity(fuel)
	if world.Components.WeightCapacity.Has(fire) && !world.Components.WeightDirty.Has(fire) {
		world.Components.WeightDirty.Add(fire, &gc.WeightDirty{})
	}
}
