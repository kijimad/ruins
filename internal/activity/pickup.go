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

	// 対象タイルのフィールドアイテムと拾えるPropを検索
	var itemsToCollect []ecs.Entity
	var propsToCollect []ecs.Entity
	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		if grid.X != target.X || grid.Y != target.Y {
			return
		}
		if !worldhelper.IsPickable(entity, world) {
			return
		}
		if entity.HasComponent(world.Components.Item) && entity.HasComponent(world.Components.LocationOnField) {
			itemsToCollect = append(itemsToCollect, entity)
		} else {
			propsToCollect = append(propsToCollect, entity)
		}
	}))

	if len(itemsToCollect) == 0 && len(propsToCollect) == 0 {
		return fmt.Errorf("拾えるものがありません")
	}

	// 収集されたアイテムを処理
	collectedCount := 0
	var errs []error
	for _, itemEntity := range itemsToCollect {
		if err := pa.collectFieldItem(actor, world, itemEntity); err != nil {
			errs = append(errs, err)
			continue
		}
		collectedCount++
	}
	for _, propEntity := range propsToCollect {
		pa.collectProp(actor, world, propEntity)
		collectedCount++
	}

	if collectedCount == 0 {
		return fmt.Errorf("拾得に失敗しました")
	}

	log.Debug("拾得完了", "count", collectedCount)

	// プレイヤーの場合のみ複数収集時の総括メッセージを表示
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

// collectProp はPropをバックパックに移動する。
// Prop系コンポーネントは保持したまま、座標だけ消してバックパックに入れる
func (pa *PickupActivity) collectProp(actor ecs.Entity, world w.World, propEntity ecs.Entity) {
	formattedName := worldhelper.GetEntityName(propEntity, world)

	worldhelper.MoveToBackpack(world, propEntity, actor)

	// フィールドから消す
	propEntity.RemoveComponent(world.Components.GridElement)

	gamelog.New(worldhelper.GetGameLog(world)).
		ItemName(formattedName).
		Append(" を拾った。").
		Log()
}

// collectFieldItem はフィールドアイテムを収集してバックパックに移動する
func (pa *PickupActivity) collectFieldItem(actor ecs.Entity, world w.World, itemEntity ecs.Entity) error {
	itemName := worldhelper.GetEntityName(itemEntity, world)

	formattedName := worldhelper.FormatItemName(world, itemEntity)

	// フィールドからバックパックに移動
	worldhelper.MoveToBackpack(world, itemEntity, actor)

	// グリッド表示コンポーネントを削除（フィールドから消す）
	if itemEntity.HasComponent(world.Components.GridElement) {
		itemEntity.RemoveComponent(world.Components.GridElement)
	}

	// 既存のバックパック内の同名Stackableアイテムを統合する処理
	if err := worldhelper.MergeInventoryItem(world, itemName); err != nil {
		return fmt.Errorf("インベントリ統合エラー: %w", err)
	}

	gamelog.New(worldhelper.GetGameLog(world)).
		ItemName(formattedName).
		Append(" を入手した。").
		Log()

	return nil
}
