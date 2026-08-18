package activity

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// PickupBehavior はBehaviorの実装
type PickupBehavior struct{}

// Info はBehaviorの実装
func (pb *PickupBehavior) Info() Info {
	return Info{
		Name:            "Pick Up",
		Description:     "Pick up an item",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.MinorActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (pb *PickupBehavior) Name() gc.BehaviorName {
	return gc.BehaviorPickup
}

// NewPickupActivity は特定のエンティティ1つを拾う拾得アクティビティを組む。
func NewPickupActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorPickup, 0)
	comp.Params = &gc.PickupParams{Targets: []ecs.Entity{target}}
	return comp
}

// NewPickupTileActivity は指定タイル上の拾得可能なエンティティを全部拾う拾得アクティビティを組む。
// 何を拾うかはここで解決して Targets に確定させる。behavior はタイル走査を持たない。
func NewPickupTileActivity(world w.World, tile consts.Coord[consts.Tile]) *gc.Activity {
	comp := NewActivity(gc.BehaviorPickup, 0)
	comp.Params = &gc.PickupParams{Targets: query.PickablesAt(world, tile)}
	return comp
}

// NewPickupStackActivity は target と同一スタックのエンティティをまとめて拾う拾得アクティビティを組む。
// 一覧の1行はスタック代表なので、代表と束ねられた全個体を Targets に確定させ、表示の個数と拾う個数を揃える。
func NewPickupStackActivity(world w.World, target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorPickup, 0)
	comp.Params = &gc.PickupParams{Targets: query.StackMembers(world, target)}
	return comp
}

// Validate はアイテム拾得アクティビティの検証を行う。1つでも拾えるものがあれば有効。
func (pb *PickupBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PickupParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	for _, entity := range p.Targets {
		if query.IsPickable(entity, world) {
			return nil
		}
	}
	return &UserError{Msg: query.T(world, "nothing to pick up")}
}

// Start はアイテム拾得開始時の処理を実行する
func (pb *PickupBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item pickup started", "actor", actor)
	return nil
}

// DoTurn はアイテム拾得アクティビティの1ターン分の処理を実行する
func (pb *PickupBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// アイテム拾得処理を実行
	if err := pb.performPickup(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("item pickup error: %s", err.Error()))
		return err
	}

	// 拾得処理完了
	Complete(comp)

	return nil
}

// Finish はアイテム拾得完了時の処理を実行する
func (pb *PickupBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item pickup activity finished", "actor", actor)
	return nil
}

// Canceled はアイテム拾得キャンセル時の処理を実行する
func (pb *PickupBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item pickup canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performPickup は Params に確定済みの Targets を拾う。同一スタックは1行にまとめてログし、
// 表示の個数と実際に拾った個数を揃える。拾得不能になったものは飛ばす。
func (pb *PickupBehavior) performPickup(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PickupParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	// 構築後に拾えなくなったものは除く。着手時に他の拾得や消滅で状態が変わりうる
	var pickable []ecs.Entity
	for _, entity := range p.Targets {
		if query.IsPickable(entity, world) {
			pickable = append(pickable, entity)
		}
	}
	if len(pickable) == 0 {
		return fmt.Errorf("nothing to pick up")
	}

	actorName := query.GetEntityName(actor, world)
	total := 0
	var errs []error
	for _, stack := range query.GroupStacks(world, pickable) {
		// 表示名と個数は移動前に確定する。移動後はスタックが割れて個数が変わるため先に数え込む
		formattedName := query.FormatItemName(world, stack.Rep)
		moved, err := lifecycle.MoveMembersToBackpack(world, stack.Members, actor)
		if err != nil {
			errs = append(errs, err)
		}
		if moved == 0 {
			continue
		}
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "%s picked up %s.",
				query.NameMarkup(actor, actorName, world),
				gamelog.Tag("item", formattedName))).
			Log()
		total += moved
	}

	if total == 0 {
		if len(errs) > 0 {
			return fmt.Errorf("some pickups failed: %w", errors.Join(errs...))
		}
		return fmt.Errorf("nothing to pick up")
	}

	log.Debug("pickup finished", "count", total)

	if len(errs) > 0 {
		return fmt.Errorf("some pickups failed: %w", errors.Join(errs...))
	}

	return nil
}
