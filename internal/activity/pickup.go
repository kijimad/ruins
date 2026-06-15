package activity

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// PickupActivity はBehaviorの実装
type PickupActivity struct{}

// Info はBehaviorの実装
func (pa *PickupActivity) Info() Info {
	return Info{
		Name:            "拾得",
		Description:     "アイテムを拾得する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 50,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (pa *PickupActivity) Name() gc.BehaviorName {
	return gc.BehaviorPickup
}

// Validate はアイテム拾得アクティビティの検証を行う
func (pa *PickupActivity) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	target, err := requireDestination(comp)
	if err != nil {
		return err
	}

	hasPickable := false
	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if hasPickable {
			return
		}
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		if grid.X != target.X || grid.Y != target.Y {
			return
		}
		if worldhelper.IsPickable(entity, world) {
			hasPickable = true
		}
	}))

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

// performPickupActivity は実際のアイテム拾得処理を実行する
func (pa *PickupActivity) performPickupActivity(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	target, err := requireDestination(comp)
	if err != nil {
		return err
	}

	// 対象タイルの拾得可能なエンティティを検索
	var toCollect []ecs.Entity
	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		if grid.X != target.X || grid.Y != target.Y {
			return
		}
		if worldhelper.IsPickable(entity, world) {
			toCollect = append(toCollect, entity)
		}
	}))

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

	if collectedCount > 1 && actor.HasComponent(world.Components.Player) {
		gamelog.New(worldhelper.GetGameLog(world)).
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
	name := worldhelper.GetEntityName(entity, world)
	formattedName := worldhelper.FormatItemName(world, entity)

	worldhelper.MoveToBackpack(world, entity, actor)
	entity.RemoveComponent(world.Components.GridElement)

	// Stackableなら同名アイテムを統合する
	if entity.HasComponent(world.Components.Stackable) {
		if err := worldhelper.MergeInventoryItem(world, name); err != nil {
			return fmt.Errorf("インベントリ統合エラー: %w", err)
		}
	}

	gamelog.New(worldhelper.GetGameLog(world)).
		ItemName(formattedName).
		Append(" を入手した。").
		Log()

	return nil
}
