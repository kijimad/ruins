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

// searchKgPerTurn は収納を探すのに1ターンかかる容量
const searchKgPerTurn = 10

// SearchBehavior はまだ探していない収納を探し、終えると収納画面を開くアクティビティの実装
type SearchBehavior struct{}

// Info はBehaviorの実装
func (b *SearchBehavior) Info() Info {
	return Info{
		Name:            "Search",
		Description:     "Search a container",
		Interruptible:   true,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (b *SearchBehavior) Name() gc.BehaviorName {
	return gc.BehaviorSearch
}

// NewSearchActivity は収納 target を探すアクティビティを組む。容量10kgごとに1ターン、最低1ターンかかる
func NewSearchActivity(target ecs.Entity, world w.World) *gc.Activity {
	turns := 1
	if world.ECS.Alive(target) && world.Components.WeightCapacity.Has(target) {
		turns = max(1, int(world.Components.WeightCapacity.Get(target).Max/consts.MilligramPerKg/searchKgPerTurn))
	}
	comp := NewActivity(gc.BehaviorSearch, turns)
	comp.Params = &gc.SearchParams{Target: target}
	return comp
}

// Validate は探す前の検証を行う
func (b *SearchBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.SearchParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !world.ECS.Alive(p.Target) || !world.Components.Unsearched.Has(p.Target) {
		return &UserError{Msg: query.T(world, "there is nothing to search")}
	}
	if !query.IsAreaSafe(world, actor) {
		return &UserError{Msg: query.T(world, "cannot search because enemies are nearby")}
	}
	return nil
}

// Start は探し始めの処理を実行する
func (b *SearchBehavior) Start(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.SearchParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "started searching %s", query.GetEntityName(p.Target, world))).
		Log()
	return nil
}

// DoTurn は探す1ターン分の処理を実行する
func (b *SearchBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.SearchParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !world.ECS.Alive(p.Target) {
		Cancel(comp, "interrupted because the container is gone")
		return nil
	}
	if !query.IsAreaSafe(world, actor) {
		Cancel(comp, "searching interrupted because enemies are nearby")
		return nil
	}
	comp.Progress.Current++
	if comp.Progress.Current >= comp.Progress.Max {
		Complete(comp)
	}
	return nil
}

// Finish は収納を探し終えた印を付け、収納画面を開く
func (b *SearchBehavior) Finish(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.SearchParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !world.ECS.Alive(p.Target) {
		return nil
	}
	if world.Components.Unsearched.Has(p.Target) {
		world.Components.Unsearched.Remove(p.Target)
	}
	if err := lifecycle.RequestStateChange(world, gc.OpenStorageEvent(p.Target)); err != nil {
		return fmt.Errorf("storage menu state change request error: %w", err)
	}
	return nil
}

// Canceled は探すのを中断したときのログを出す。対象が消えての中断もあるので、名前は対象が残るときだけ出す
func (b *SearchBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if !world.Components.Player.Has(actor) {
		return nil
	}
	message := query.T(world, "interrupted searching")
	if p, ok := comp.Params.(*gc.SearchParams); ok && world.ECS.Alive(p.Target) {
		message = query.T(world, "interrupted searching %s", query.GetEntityName(p.Target, world))
	}
	gamelog.New(query.GetGameLog(world)).
		Markup(message).
		Log()
	return nil
}
