package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// LightLit は light が今照らしているかを返す。点いていて、残量を持つなら残っていて、照らせる置き場所にあるときに照らす。
// 照らせる置き場所は、位置を持つ床や置物と、player の手持ち欄で、背負っている物は照らさない
func LightLit(world w.World, player, light ecs.Entity) bool {
	if !world.Components.LightSource.Has(light) || !world.Components.LightSource.Get(light).Enabled {
		return false
	}
	if world.Components.Charge.Has(light) && world.Components.Charge.Get(light).Current <= 0 {
		return false
	}
	if world.Components.GridElement.Has(light) {
		return true
	}
	if !world.Components.LocationEquipped.Has(light) {
		return false
	}
	equipped := world.Components.LocationEquipped.Get(light)
	if equipped.Owner != player {
		return false
	}
	_, ok := equipped.EquipmentSlot.HandIndex()
	return ok
}
