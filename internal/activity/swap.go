package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// SwapBehavior はBehaviorの実装
type SwapBehavior struct{}

// Info はBehaviorの実装
func (sb *SwapBehavior) Info() Info {
	return Info{
		Name:            "Swap",
		Description:     "Swap places with an adjacent companion",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (sb *SwapBehavior) Name() gc.BehaviorName {
	return gc.BehaviorSwap
}

// NewSwapActivity は入れ替える相手を指定して入れ替えアクティビティを組む。
func NewSwapActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorSwap, 0)
	comp.Params = &gc.SwapParams{Target: target}
	return comp
}

// Validate は入れ替えアクティビティの検証を行う
func (sb *SwapBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.SwapParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	// ゼロ値・死亡エンティティはArkのHasでパニックするため先に弾く
	if !world.ECS.Alive(p.Target) {
		return fmt.Errorf("swap target is not alive")
	}
	if !world.Components.GridElement.Has(actor) || !world.Components.GridElement.Has(p.Target) {
		return ErrGridElementNotFound
	}
	return nil
}

// Start はBehaviorの実装
func (sb *SwapBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("swap started", "actor", actor)
	return nil
}

// DoTurn は actor と相手の座標を入れ替える
func (sb *SwapBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.SwapParams)
	if !ok {
		Cancel(comp, "swap target is not set")
		return ErrParamsTypeMismatch
	}
	actorGrid := world.Components.GridElement.Get(actor)
	targetGrid := world.Components.GridElement.Get(p.Target)
	actorGrid.Coord, targetGrid.Coord = targetGrid.Coord, actorGrid.Coord
	// 2体が互いの座標へ移るので増分更新でなく再構築させる
	query.InvalidateSpatialIndex(world)

	Complete(comp)
	return nil
}

// Finish はプレイヤーの移動先のタイルイベントを表示する
func (sb *SwapBehavior) Finish(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	if world.Components.Player.Has(actor) {
		showTileInteractionMessage(world, world.Components.GridElement.Get(actor))
	}
	return nil
}

// Canceled はBehaviorの実装
func (sb *SwapBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("swap canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}
