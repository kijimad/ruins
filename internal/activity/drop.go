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

// DropActivity はBehaviorの実装
type DropActivity struct {
	Target      ecs.Entity
	Destination gc.GridElement
}

// Info はBehaviorの実装
func (da *DropActivity) Info() Info {
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
func (da *DropActivity) Name() gc.BehaviorName {
	return gc.BehaviorDrop
}

// BuildActivity はBehaviorの実装
func (da *DropActivity) BuildActivity(_ ecs.Entity, _ w.World) (*gc.Activity, error) {
	comp, err := NewActivity(da, 1)
	if err != nil {
		return nil, err
	}
	comp.Target = &da.Target
	comp.Destination = &da.Destination
	return comp, nil
}

// Validate はアイテムドロップアクティビティの検証を行う
func (da *DropActivity) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
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
func (da *DropActivity) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテムドロップ開始", "actor", actor, "target", *comp.Target)
	return nil
}

// DoTurn はアイテムドロップアクティビティの1ターン分の処理を実行する
func (da *DropActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// アイテムドロップ処理を実行
	if err := da.performDropActivity(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("アイテムドロップエラー: %s", err.Error()))
		return err
	}

	// ドロップ処理完了
	Complete(comp)
	return nil
}

// Finish はアイテムドロップ完了時の処理を実行する
func (da *DropActivity) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテムドロップアクティビティ完了", "actor", actor)
	return nil
}

// Canceled はアイテムドロップキャンセル時の処理を実行する
func (da *DropActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテムドロップキャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performDropActivity は実際のアイテムドロップ処理を実行する
func (da *DropActivity) performDropActivity(comp *gc.Activity, actor ecs.Entity, world w.World) error {
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
