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

// TransferActivity はエンティティ間でアイテムを転送するBehavior実装。
// Targetに転送するアイテム、Recipientに受取人を指定する
type TransferActivity struct {
	Target    ecs.Entity
	Recipient ecs.Entity
}

// Info はBehaviorの実装
func (ta *TransferActivity) Info() Info {
	return Info{
		Name:            "転送",
		Description:     "アイテムを他のエンティティに渡す",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.MinorActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (ta *TransferActivity) Name() gc.BehaviorName {
	return gc.BehaviorTransfer
}

// BuildActivity はBehaviorの実装
func (ta *TransferActivity) BuildActivity(_ ecs.Entity, _ w.World) (*gc.Activity, error) {
	comp, err := NewActivity(ta, 1)
	if err != nil {
		return nil, err
	}
	comp.Target = &ta.Target
	comp.Recipient = &ta.Recipient
	return comp, nil
}

// Validate はアイテム転送アクティビティの検証を行う
func (ta *TransferActivity) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("転送対象が指定されていません")
	}
	if comp.Recipient == nil {
		return fmt.Errorf("受取人が指定されていません")
	}

	target := *comp.Target
	if !world.Components.LocationInBackpack.Has(target) {
		return fmt.Errorf("アイテムがバックパック内にありません")
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

// performTransfer はアイテムを受取人のバックパックに移動する
func (ta *TransferActivity) performTransfer(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	item := *comp.Target
	recipient := *comp.Recipient

	formattedName := query.FormatItemName(world, item)
	actorName := query.GetEntityName(actor, world)
	recipientName := query.GetEntityName(recipient, world)

	if err := lifecycle.MoveToBackpack(world, item, recipient); err != nil {
		return fmt.Errorf("アイテム転送に失敗: %w", err)
	}

	logger := gamelog.New(query.GetGameLog(world))
	query.AppendNameWithColor(logger, actor, actorName, world)
	logger.
		Append(" は ").
		ItemName(formattedName).
		Append(" を ").
		Append(recipientName).
		Append(" に渡した。").
		Log()

	return nil
}
