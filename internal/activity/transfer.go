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
// Targetに転送するアイテム、Recipientに受取人を指定する。
// Singleが真なら stackable でも1個だけ渡す。補給で隊員が共有プールから1食ぶんだけ引くために使う
type TransferActivity struct {
	Target    ecs.Entity
	Recipient ecs.Entity
	Single    bool
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
func (ta *TransferActivity) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	if err := ta.performTransfer(comp, world); err != nil {
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
func (ta *TransferActivity) performTransfer(comp *gc.Activity, world w.World) error {
	item := *comp.Target
	recipient := *comp.Recipient

	// 渡す主体はアクターでなくアイテムの現所有者にする。補給ではアクターの隊員がリーダーの
	// プールから引くため、アクターを主体にすると「隊員は隊員に渡した」と自己転送の誤ログになる。
	giver := world.Components.LocationInBackpack.Get(item).Owner
	giverName := query.GetEntityName(giver, world)
	recipientName := query.GetEntityName(recipient, world)

	// ログ名は転送前に確定させる。Single では1個だけ渡すので個数を出さず素の名前にする。
	// 丸ごと転送では移す個数がそのまま渡るので FormatItemName の「(N個)」表記が正しい。
	itemName := query.FormatItemName(world, item)
	if ta.Single {
		if nameComp := world.Components.Name.Get(item); nameComp != nil {
			itemName = nameComp.Name
		}
	}

	if ta.Single {
		if err := lifecycle.TransferOneUnit(world, item, recipient); err != nil {
			return fmt.Errorf("アイテム転送に失敗: %w", err)
		}
	} else {
		if err := lifecycle.MoveToBackpack(world, item, recipient); err != nil {
			return fmt.Errorf("アイテム転送に失敗: %w", err)
		}
	}

	logger := gamelog.New(query.GetGameLog(world))
	query.AppendNameWithColor(logger, giver, giverName, world)
	logger.
		Append(" は ").
		ItemName(itemName).
		Append(" を ").
		Append(recipientName).
		Append(" に渡した。").
		Log()

	return nil
}
