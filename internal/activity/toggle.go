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

// ToggleBehavior は持っている物のスイッチを点け消しする行動
type ToggleBehavior struct{}

// Info はBehaviorの実装
func (tb *ToggleBehavior) Info() Info {
	return Info{
		Name:            "Toggle",
		Description:     "Switch an item on or off",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.MinorActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (tb *ToggleBehavior) Name() gc.BehaviorName {
	return gc.BehaviorToggle
}

// NewToggleActivity は点け消しする物を指定して点け消しのアクティビティを組む
func NewToggleActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorToggle, 0)
	comp.Params = &gc.ConsumeParams{Target: target}
	return comp
}

// Validate は点け消しする物がスイッチを持ち、actor が持っていることを確かめる
func (tb *ToggleBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ConsumeParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !world.Components.Switch.Has(p.Target) {
		return fmt.Errorf("item has no switch")
	}
	if !query.CarriedBy(world, p.Target, actor) {
		return fmt.Errorf("item is not carried")
	}
	return nil
}

// Start はBehaviorの実装
func (tb *ToggleBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	if p, ok := comp.Params.(*gc.ConsumeParams); ok {
		log.Debug("toggle started", "actor", actor, "item", p.Target)
	}
	return nil
}

// DoTurn はスイッチを切り替え、持ち主の灯りと視界を計算し直させる
func (tb *ToggleBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.ConsumeParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	sw := world.Components.Switch.Get(p.Target)
	sw.On = !sw.On

	name := gamelog.Tag("item", query.GetEntityName(p.Target, world))
	text := query.T(world, "You switched off %s.", name)
	if sw.On {
		text = query.T(world, "You switched on %s.", name)
	}
	gamelog.New(query.GetGameLog(world)).Markup(text).Log()
	// 尽きた物はオンでも働かない。補充すればそのまま働くので、点けることは許して理由だけ知らせる
	if sw.On && world.Components.Charge.Has(p.Target) && world.Components.Charge.Get(p.Target).Current <= 0 {
		gamelog.New(query.GetGameLog(world)).Markup(query.T(world, "%s has no fuel.", name)).Log()
	}

	lifecycle.MarkStatsChanged(world, actor)
	query.GetVisionState(world).RequestUpdate()
	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (tb *ToggleBehavior) Finish(_ *gc.Activity, _ ecs.Entity, _ w.World) error {
	return nil
}

// Canceled はBehaviorの実装
func (tb *ToggleBehavior) Canceled(_ *gc.Activity, _ ecs.Entity, _ w.World) error {
	return nil
}
