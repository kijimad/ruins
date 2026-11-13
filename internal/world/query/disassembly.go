package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ItemDisassemblyYield は item を分解したときに戻る素材の id と個数を返す。分解できなければ ok は false
func ItemDisassemblyYield(world w.World, item ecs.Entity) (id string, count int, ok bool) {
	if !world.ECS.Alive(item) || !world.Components.RawID.Has(item) || !world.Components.Material.Has(item) || !world.Components.Weight.Has(item) {
		return "", 0, false
	}
	return disassemblyYield(world,
		world.Components.RawID.Get(item).ID,
		world.Components.Material.Get(item).Kind,
		world.Components.Weight.Get(item).Milligram,
		world.Components.Deployable.Has(item),
	)
}

// SpecDisassemblyYield は生成前のアイテムの spec を分解したときに戻る素材の id と個数を返す。分解できなければ ok は false
func SpecDisassemblyYield(world w.World, spec gc.EntitySpec) (id string, count int, ok bool) {
	if spec.RawID == nil || spec.Material == nil || spec.Weight == nil {
		return "", 0, false
	}
	return disassemblyYield(world, spec.RawID.ID, spec.Material.Kind, spec.Weight.Milligram, spec.Deployable != nil)
}

// disassemblyYield は装着する設備を除き、材質と重さから戻る素材を導く
func disassemblyYield(world w.World, id string, material rawschema.Material, weight consts.Milligram, deployable bool) (string, int, bool) {
	if deployable {
		return "", 0, false
	}
	return world.Resources.MaterialDefs.DisassemblyOf(id, material, weight)
}
