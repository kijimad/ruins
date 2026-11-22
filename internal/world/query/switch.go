package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// SwitchMarker はスイッチを持つ物の、オンかオフかの訳した表示を返す。スイッチを持たなければ空文字
func SwitchMarker(world w.World, entity ecs.Entity) string {
	if !world.Components.Switch.Has(entity) {
		return ""
	}
	if world.Components.Switch.Get(entity).On {
		return T(world, "Switched on")
	}
	return T(world, "Switched off")
}
