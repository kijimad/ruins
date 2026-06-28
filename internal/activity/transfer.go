package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TransferActivity は隊員がバックパック内のアイテムをリーダーに転送するBehavior実装
type TransferActivity struct{}

// Info はBehaviorの実装
func (ta *TransferActivity) Info() Info {
	return Info{
		Name:            "転送",
		Description:     "アイテムをリーダーに渡す",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 50,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (ta *TransferActivity) Name() gc.BehaviorName {
	return gc.BehaviorTransfer
}

// Validate はアイテム転送アクティビティの検証を行う
func (ta *TransferActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("転送対象が指定されていません")
	}

	target := *comp.Target
	if !target.HasComponent(world.Components.LocationInBackpack) {
		return fmt.Errorf("アイテムがバックパック内にありません")
	}

	if !actor.HasComponent(world.Components.SquadMember) {
		return fmt.Errorf("隊員のみアイテム転送を実行できます")
	}

	return nil
}

// Start はアイテム転送開始時の処理を実行する
func (ta *TransferActivity) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム転送開始", "actor", actor)
	return nil
}

// DoTurn はアイテム転送アクティビティの1ターン分の処理を実行する
func (ta *TransferActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if err := ta.performTransfer(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("アイテム転送エラー: %s", err.Error()))
		return err
	}

	Complete(comp)
	return nil
}

// Finish はアイテム転送完了時の処理を実行する
func (ta *TransferActivity) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム転送アクティビティ完了", "actor", actor)
	return nil
}

// Canceled はアイテム転送キャンセル時の処理を実行する
func (ta *TransferActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム転送キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performTransfer はアイテムをリーダーのバックパックに移動する
func (ta *TransferActivity) performTransfer(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	item := *comp.Target
	if !item.HasComponent(world.Components.LocationInBackpack) {
		return fmt.Errorf("アイテムがバックパック内にありません")
	}

	sm := world.Components.SquadMember.Get(actor).(*gc.SquadMember)
	leader := sm.Leader

	formattedName := query.FormatItemName(world, item)

	if err := lifecycle.MoveToBackpack(world, item, leader); err != nil {
		return fmt.Errorf("リーダーへの転送に失敗: %w", err)
	}

	actorName := ""
	if actor.HasComponent(world.Components.Name) {
		actorName = world.Components.Name.Get(actor).(*gc.Name).Name
	}
	gamelog.New(query.GetGameLog(world)).
		Append(actorName + " が ").
		ItemName(formattedName).
		Append(" を渡した。").
		Log()

	return nil
}
