package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ItemDisassemblyYield は item を分解したときに戻る素材の id と個数を返す。装着する設備と、材質から素材を導けない物は ok が false
func ItemDisassemblyYield(world w.World, item ecs.Entity) (id string, count int, ok bool) {
	if !world.ECS.Alive(item) || !world.Components.RawID.Has(item) || !world.Components.Material.Has(item) || !world.Components.Weight.Has(item) {
		return "", 0, false
	}
	if world.Components.Deployable.Has(item) {
		return "", 0, false
	}
	return world.Resources.MaterialDefs.DisassemblyOf(
		GetEntityID(item, world),
		world.Components.Material.Get(item).Kind,
		world.Components.Weight.Get(item).Milligram,
	)
}
