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
		Name:            "Drop",
		Description:     "Place an item at your feet",
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
	comp.Params = &gc.PlaceParams{Target: target, Destination: destination}
	return comp
}

// Validate はアイテムドロップアクティビティの検証を行う
func (db *DropBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	target := p.Target

	// Targetがバックパック内にあることを確認する
	if !world.Components.LocationInBackpack.Has(target) {
		return fmt.Errorf("item is not in the backpack")
	}

	// 配置先タイル座標を取得できるか確認する
	if _, err := requireDestination(comp); err != nil {
		return err
	}

	return nil
}

// Start はアイテムドロップ開始時の処理を実行する
func (db *DropBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	if p, ok := comp.Params.(*gc.PlaceParams); ok {
		log.Debug("item drop started", "actor", actor, "target", p.Target)
	}
	return nil
}

// DoTurn はアイテムドロップアクティビティの1ターン分の処理を実行する
func (db *DropBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// アイテムドロップ処理を実行
	if err := db.performDrop(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("item drop error: %s", err.Error()))
		return err
	}

	// ドロップ処理完了
	Complete(comp)
	return nil
}

// Finish はアイテムドロップ完了時の処理を実行する
func (db *DropBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item drop activity finished", "actor", actor)
	return nil
}

// Canceled はアイテムドロップキャンセル時の処理を実行する
func (db *DropBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item drop canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performDrop は実際のアイテムドロップ処理を実行する
func (db *DropBehavior) performDrop(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.PlaceParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	targetTile, err := requireDestination(comp)
	if err != nil {
		return err
	}

	target := p.Target
	// 名前は移動前に確定する。移動後はスタックが割れて個数が変わるため、先に N を数え込む
	formattedName := query.FormatItemName(world, target)

	// 一覧の1行はスタック代表なので、同一スタックのエンティティをまとめて足元へ落とす
	lifecycle.MoveMembersToField(world, query.StackMembers(world, target), targetTile, actor)

	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "Dropped %s.", gamelog.Tag("item", formattedName))).
		Log()

	return nil
}
