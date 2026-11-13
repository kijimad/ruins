package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// DisassembleBehavior は所持品のアイテムを材質の素材に戻すアクティビティの実装
type DisassembleBehavior struct{}

// Info はBehaviorの実装
func (b *DisassembleBehavior) Info() Info {
	return Info{
		Name:            "Disassemble",
		Description:     "Break an item down into materials",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (b *DisassembleBehavior) Name() gc.BehaviorName {
	return gc.BehaviorDisassemble
}

// NewDisassembleActivity は分解するアイテムを指定して分解アクティビティを組む。
// 重さ1kgごとに1ターンかかり、最低1ターンにする
func NewDisassembleActivity(target ecs.Entity, world w.World) *gc.Activity {
	turns := 1
	if world.ECS.Alive(target) && world.Components.Weight.Has(target) {
		turns = max(1, int(world.Components.Weight.Get(target).Milligram/consts.MilligramPerKg))
	}
	comp := NewActivity(gc.BehaviorDisassemble, turns)
	comp.Params = &gc.DisassembleParams{Target: target}
	return comp
}

// DisassemblyYield は item を分解したときに戻る素材の id と個数を返す。分解できなければ ok は false
func DisassemblyYield(world w.World, item ecs.Entity) (id string, count int, ok bool) {
	if !world.ECS.Alive(item) || !world.Components.Material.Has(item) || !world.Components.Weight.Has(item) {
		return "", 0, false
	}
	if world.Components.Deployable.Has(item) {
		return "", 0, false
	}
	return world.Resources.MaterialDefs.DisassemblyOf(
		query.GetEntityID(item, world),
		world.Components.Material.Get(item).Kind,
		world.Components.Weight.Get(item).Milligram,
	)
}

// inBackpackOf は item が actor の所持品にあるかを返す
func inBackpackOf(world w.World, item, actor ecs.Entity) bool {
	return world.ECS.Alive(item) &&
		world.Components.LocationInBackpack.Has(item) &&
		world.Components.LocationInBackpack.Get(item).Owner == actor
}

// Validate は分解の検証を行う
func (b *DisassembleBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.DisassembleParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !inBackpackOf(world, p.Target, actor) {
		return &UserError{Msg: query.T(world, "the item is not in your inventory")}
	}
	if _, _, ok := DisassemblyYield(world, p.Target); !ok {
		return &UserError{Msg: query.T(world, "this item cannot be disassembled")}
	}
	if !query.IsAreaSafe(world, actor) {
		return &UserError{Msg: query.T(world, "cannot disassemble because enemies are nearby")}
	}
	return nil
}

// Start は分解開始時の処理を実行する
func (b *DisassembleBehavior) Start(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.DisassembleParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "started disassembling %s", query.GetEntityName(p.Target, world))).
		Log()
	return nil
}

// DoTurn は分解の1ターン分の処理を実行する
func (b *DisassembleBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.DisassembleParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !inBackpackOf(world, p.Target, actor) {
		Cancel(comp, "interrupted because the item is gone")
		return nil
	}
	if !query.IsAreaSafe(world, actor) {
		Cancel(comp, "disassembly interrupted because enemies are nearby")
		return nil
	}
	comp.Progress.Current++
	if comp.Progress.Current >= comp.Progress.Max {
		Complete(comp)
	}
	return nil
}

// Finish はアイテムを消し、戻る素材を所持品に足す
func (b *DisassembleBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.DisassembleParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !inBackpackOf(world, p.Target, actor) {
		return nil
	}
	id, count, ok := DisassemblyYield(world, p.Target)
	if !ok {
		return nil
	}
	name := query.GetEntityName(p.Target, world)
	if err := lifecycle.ChangeItemCount(world, p.Target, -1); err != nil {
		return fmt.Errorf("failed to consume disassembled item: %w", err)
	}
	material, err := lifecycle.SpawnBackpackItem(world, id, count)
	if err != nil {
		return fmt.Errorf("failed to spawn disassembled material: %w", err)
	}
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "disassembled %s into %s x%d", name, query.GetEntityName(material, world), count)).
		Log()
	return nil
}

// Canceled は分解キャンセル時の処理を実行する。対象が消えたことによる中断もあるため、名前は対象が残っている場合だけ出す
func (b *DisassembleBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if !world.Components.Player.Has(actor) {
		return nil
	}
	message := query.T(world, "interrupted disassembling")
	if p, ok := comp.Params.(*gc.DisassembleParams); ok && world.ECS.Alive(p.Target) {
		message = query.T(world, "interrupted disassembling %s", query.GetEntityName(p.Target, world))
	}
	gamelog.New(query.GetGameLog(world)).
		Markup(message).
		Log()
	return nil
}
