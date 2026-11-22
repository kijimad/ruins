package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// LightLit は light が今照らしているかを返す。点いていて、スイッチを持つならオンで、残量が残り、
// 床か holder の持ち物にあるときに照らす。視界と減らす処理はプレイヤーを holder に渡す
func LightLit(world w.World, holder, light ecs.Entity) bool {
	if !world.Components.LightSource.Has(light) || !world.Components.LightSource.Get(light).Enabled {
		return false
	}
	if world.Components.Switch.Has(light) && !IsSwitchedOn(world, light) {
		return false
	}
	if world.Components.Charge.Has(light) && world.Components.Charge.Get(light).Current <= 0 {
		return false
	}
	if world.Components.GridElement.Has(light) {
		return true
	}
	return CarriedBy(world, light, holder)
}

// CarriedBy は item が owner のバックパックか装備にあるかを返す
func CarriedBy(world w.World, item, owner ecs.Entity) bool {
	carrier, ok := Carrier(world, item)
	return ok && carrier == owner
}

// Carrier は item を持つ者を返す。バックパックか装備になければ ok は偽
func Carrier(world w.World, item ecs.Entity) (ecs.Entity, bool) {
	switch {
	case world.Components.LocationInBackpack.Has(item):
		return world.Components.LocationInBackpack.Get(item).Owner, true
	case world.Components.LocationEquipped.Has(item):
		return world.Components.LocationEquipped.Get(item).Owner, true
	}
	return ecs.Entity{}, false
}
