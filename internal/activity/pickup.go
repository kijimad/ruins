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

// NewPickupActivity は拾得対象または拾得先を指定して拾得アクティビティを組む。
// target が nil なら足元や指定座標から拾う。
func NewPickupActivity(target *ecs.Entity, destination *gc.GridElement) *gc.Activity {
	comp := NewActivity(gc.BehaviorPickup, 0)
	// Target 未指定は足元拾得を表すので、無効エンティティを標識に置く
	p := &gc.PlaceParams{Target: gc.InvalidEntity}
	if target != nil {
		p.Target = *target
	}
	if destination != nil {
		p.Destination = *destination
	}
	comp.Params = p
	return comp
}

// Validate はアイテム拾得アクティビティの検証を行う
func (pb *PickupBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return fmt.Errorf("拾得対象が指定されていません")
	}

	// Targetが指定されている場合は、そのエンティティが拾得可能かだけを確認する
	if p.Target != gc.InvalidEntity {
		if !query.IsPickable(p.Target, world) {
			return fmt.Errorf("拾えるものがありません")
		}
		return nil
	}

	target, err := requireDestination(comp)
	if err != nil {
		return err
	}

	hasPickable := false
	pickableQuery := query.ActiveFilter1[gc.GridElement](world).Query()
	for pickableQuery.Next() {
		entity := pickableQuery.Entity()
		if hasPickable {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		if grid.X != target.X || grid.Y != target.Y {
			continue
		}
		if query.IsPickable(entity, world) {
			hasPickable = true
		}
	}

	if !hasPickable {
		return fmt.Errorf("拾えるものがありません")
	}

	return nil
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

// performPickup は実際のアイテム拾得処理を実行する。
// Targetが指定されている場合はそのエンティティだけを拾い、
// 未指定の場合はDestinationタイル上の全拾得可能エンティティを拾う
func (pb *PickupBehavior) performPickup(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return fmt.Errorf("拾得対象が指定されていません")
	}

	// Targetが指定されている場合は、そのエンティティだけを拾う
	if p.Target != gc.InvalidEntity {
		if !query.IsPickable(p.Target, world) {
			return fmt.Errorf("拾えるものがありません")
		}
		return pb.collect(actor, world, p.Target)
	}

	target, err := requireDestination(comp)
	if err != nil {
		return err
	}

	// 対象タイルの拾得可能なエンティティを検索
	var toCollect []ecs.Entity
	collectQuery := query.ActiveFilter1[gc.GridElement](world).Query()
	for collectQuery.Next() {
		entity := collectQuery.Entity()
		grid := world.Components.GridElement.Get(entity)
		if grid.X != target.X || grid.Y != target.Y {
			continue
		}
		if query.IsPickable(entity, world) {
			toCollect = append(toCollect, entity)
		}
	}

	if len(toCollect) == 0 {
		return fmt.Errorf("拾えるものがありません")
	}

	collectedCount := 0
	var errs []error
	for _, entity := range toCollect {
		if err := pb.collect(actor, world, entity); err != nil {
			errs = append(errs, err)
			continue
		}
		collectedCount++
	}

	if collectedCount == 0 {
		return fmt.Errorf("拾得に失敗しました")
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
