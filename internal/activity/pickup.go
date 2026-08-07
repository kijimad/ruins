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
		Name:            "拾得",
		Description:     "アイテムを拾得する",
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

// Validate はアイテム拾得アクティビティの検証を行う。1つでも拾えるものがあれば有効。
func (pb *PickupBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PickupParams)
	if !ok {
		return fmt.Errorf("拾得対象が指定されていません")
	}
	for _, entity := range p.Targets {
		if query.IsPickable(entity, world) {
			return nil
		}
	}
	return fmt.Errorf("拾えるものがありません")
}

// Start はアイテム拾得開始時の処理を実行する
func (pb *PickupBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム拾得開始", "actor", actor)
	return nil
}

// DoTurn はアイテム拾得アクティビティの1ターン分の処理を実行する
func (pb *PickupBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// アイテム拾得処理を実行
	if err := pb.performPickup(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("アイテム拾得エラー: %s", err.Error()))
		return err
	}

	// 拾得処理完了
	Complete(comp)

	return nil
}

// Finish はアイテム拾得完了時の処理を実行する
func (pb *PickupBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム拾得アクティビティ完了", "actor", actor)
	return nil
}

// Canceled はアイテム拾得キャンセル時の処理を実行する
func (pb *PickupBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム拾得キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performPickup は Params に確定済みの Targets を順に拾う。拾得不能になったものは飛ばす。
func (pb *PickupBehavior) performPickup(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PickupParams)
	if !ok {
		return fmt.Errorf("拾得対象が指定されていません")
	}

	collectedCount := 0
	var errs []error
	for _, entity := range p.Targets {
		// 構築後に拾えなくなったものは飛ばす。着手時に他の拾得や消滅で状態が変わりうる
		if !query.IsPickable(entity, world) {
			continue
		}
		if err := pb.collect(actor, world, entity); err != nil {
			errs = append(errs, err)
			continue
		}
		collectedCount++
	}

	if collectedCount == 0 {
		if len(errs) > 0 {
			return fmt.Errorf("一部の拾得に失敗: %w", errors.Join(errs...))
		}
		return fmt.Errorf("拾えるものがありません")
	}

	log.Debug("拾得完了", "count", collectedCount)

	if collectedCount > 1 && world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Append(fmt.Sprintf("%d個を入手した", collectedCount)).
			Log()
	}

	if len(errs) > 0 {
		return fmt.Errorf("一部の拾得に失敗: %w", errors.Join(errs...))
	}

	return nil
}

// collect はフィールド上のエンティティをバックパックに移動する
func (pb *PickupBehavior) collect(actor ecs.Entity, world w.World, entity ecs.Entity) error {
	// MoveToBackpack内のmergeでentityが削除される可能性があるため、名前を先に取得する
	formattedName := query.FormatItemName(world, entity)
	actorName := query.GetEntityName(actor, world)

	if err := lifecycle.MoveToBackpack(world, entity, actor); err != nil {
		return fmt.Errorf("バックパックへの移動に失敗: %w", err)
	}
	logger := gamelog.New(query.GetGameLog(world))
	query.AppendNameWithColor(logger, actor, actorName, world)
	logger.
		Append(" は ").
		ItemName(formattedName).
		Append(" を入手した。").
		Log()

	return nil
}
