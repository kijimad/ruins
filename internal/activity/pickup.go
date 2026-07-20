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

// PickupActivity はBehaviorの実装
type PickupActivity struct {
	Target      *ecs.Entity
	Destination *gc.GridElement
}

// Info はBehaviorの実装
func (pa *PickupActivity) Info() Info {
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
func (pa *PickupActivity) Name() gc.BehaviorName {
	return gc.BehaviorPickup
}

// BuildActivity はBehaviorの実装
func (pa *PickupActivity) BuildActivity(_ ecs.Entity, _ w.World) (*gc.Activity, error) {
	comp, err := NewActivity(pa, 1)
	if err != nil {
		return nil, err
	}
	if pa.Target != nil {
		comp.Target = pa.Target
	}
	if pa.Destination != nil {
		comp.Destination = pa.Destination
	}
	return comp, nil
}

// Validate はアイテム拾得アクティビティの検証を行う
func (pa *PickupActivity) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	// Targetが指定されている場合は、そのエンティティが拾得可能かだけを確認する
	if comp.Target != nil {
		if !query.IsPickable(*comp.Target, world) {
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
func (pa *PickupActivity) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム拾得開始", "actor", actor)
	return nil
}

// DoTurn はアイテム拾得アクティビティの1ターン分の処理を実行する
func (pa *PickupActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// アイテム拾得処理を実行
	if err := pa.performPickupActivity(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("アイテム拾得エラー: %s", err.Error()))
		return err
	}

	// 拾得処理完了
	Complete(comp)

	return nil
}

// Finish はアイテム拾得完了時の処理を実行する
func (pa *PickupActivity) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム拾得アクティビティ完了", "actor", actor)
	return nil
}

// Canceled はアイテム拾得キャンセル時の処理を実行する
func (pa *PickupActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム拾得キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performPickupActivity は実際のアイテム拾得処理を実行する。
// Targetが指定されている場合はそのエンティティだけを拾い、
// 未指定の場合はDestinationタイル上の全拾得可能エンティティを拾う
func (pa *PickupActivity) performPickupActivity(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// Targetが指定されている場合は、そのエンティティだけを拾う
	if comp.Target != nil {
		if !query.IsPickable(*comp.Target, world) {
			return fmt.Errorf("拾えるものがありません")
		}
		return pa.collect(actor, world, *comp.Target)
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
		if err := pa.collect(actor, world, entity); err != nil {
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
func (pa *PickupActivity) collect(actor ecs.Entity, world w.World, entity ecs.Entity) error {
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
