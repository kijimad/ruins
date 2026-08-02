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

// DropBehavior はBehaviorの実装
type DropBehavior struct{}

// Info はBehaviorの実装
func (db *DropBehavior) Info() Info {
	return Info{
		Name:            "ドロップ",
		Description:     "アイテムを足元に置く",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.MinorActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (db *DropBehavior) Name() gc.BehaviorName {
	return gc.BehaviorDrop
}

// NewDropActivity は対象アイテムと落とす先を指定してドロップアクティビティを組む。
func NewDropActivity(target ecs.Entity, destination gc.GridElement) *gc.Activity {
	comp := NewActivity(gc.BehaviorDrop, 0)
	comp.Target = &target
	comp.Destination = &destination
	return comp
}

// Validate はアイテムドロップアクティビティの検証を行う
func (db *DropBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("ドロップ対象が指定されていません")
	}

	target := *comp.Target

	// Targetがバックパック内にあることを確認する
	if !world.Components.LocationInBackpack.Has(target) {
		return fmt.Errorf("アイテムがバックパック内にありません")
	}

	// 配置先タイル座標を取得できるか確認する
	if _, err := requireDestination(comp); err != nil {
		return err
	}

	return nil
}

// Start はアイテムドロップ開始時の処理を実行する
func (db *DropBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテムドロップ開始", "actor", actor, "target", *comp.Target)
	return nil
}

// DoTurn はアイテムドロップアクティビティの1ターン分の処理を実行する
func (db *DropBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// アイテムドロップ処理を実行
	if err := db.performDrop(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("アイテムドロップエラー: %s", err.Error()))
		return err
	}

	// ドロップ処理完了
	Complete(comp)
	return nil
}

// Finish はアイテムドロップ完了時の処理を実行する
func (db *DropBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテムドロップアクティビティ完了", "actor", actor)
	return nil
}

// Canceled はアイテムドロップキャンセル時の処理を実行する
func (db *DropBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテムドロップキャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performDrop は実際のアイテムドロップ処理を実行する
func (db *DropBehavior) performDrop(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	targetTile, err := requireDestination(comp)
	if err != nil {
		return err
	}

	target := *comp.Target
	formattedName := query.FormatItemName(world, target)

	lifecycle.MoveToField(world, target, &actor)
	world.Components.GridElement.Add(target, &gc.GridElement{Coord: targetTile})

	gamelog.New(query.GetGameLog(world)).
		ItemName(formattedName).
		Append(" を置いた。").
		Log()

	return nil
}
