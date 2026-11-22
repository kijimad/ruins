package lifecycle

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// burnFuel は燃やした item を片付ける。燃料の行は中身だけが燃え、入れ物が残量0で残る。ほかの物は消える
func burnFuel(world w.World, item ecs.Entity) {
	if _, fuel := query.FuelKindOf(world, item); fuel {
		world.Components.Charge.Get(item).Current = 0
		return
	}
	RemoveItem(world, item)
}
