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

// TransferBehavior はエンティティ間でアイテムを転送するBehavior実装。
// Targetに転送するアイテム、Recipientに受取人を指定する。
// Countは渡す個数。在庫数以上を指定すればスタックごとまとめて渡り、少なく指定すればその分だけ分割して渡す。
// 補給で共有プールから1食ぶんだけ引くときは1、丸ごと渡すときは在庫数を指定する
type TransferBehavior struct{}

// Info はBehaviorの実装
func (tb *TransferBehavior) Info() Info {
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
func (tb *TransferBehavior) Name() gc.BehaviorName {
	return gc.BehaviorTransfer
}

// NewTransferActivity は転送アイテム・受取人・個数を指定して転送アクティビティを組む。
// count が0以下なら在庫全量を渡す。個数は TransferParams.Count に持たせ、継続処理でも読める。
func NewTransferActivity(target, recipient ecs.Entity, count int) *gc.Activity {
	comp := NewActivity(gc.BehaviorTransfer, 0)
	comp.Params = &gc.TransferParams{Target: target, Recipient: recipient, Count: count}
	return comp
}

// Validate はアイテム転送アクティビティの検証を行う
func (tb *TransferBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.TransferParams)
	if !ok {
		return fmt.Errorf("転送対象が指定されていません")
	}
	// 値型のパラメータでは未指定が無効エンティティになる。ArkのHasはゼロ値でパニックするため、
	// Aliveで存在を確かめてから所持判定へ進む
	if !world.ECS.Alive(p.Target) {
		return fmt.Errorf("転送対象が指定されていません")
	}
	if !world.ECS.Alive(p.Recipient) {
		return fmt.Errorf("受取人が指定されていません")
	}

	target := p.Target
	if !world.Components.LocationInBackpack.Has(target) {
		return fmt.Errorf("アイテムがバックパック内にありません")
	}

	return nil
}

// Start はアイテム転送開始時の処理を実行する
func (tb *TransferBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム転送開始", "actor", actor)
	return nil
}

// DoTurn はアイテム転送アクティビティの1ターン分の処理を実行する
func (tb *TransferBehavior) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	if err := tb.performTransfer(comp, world); err != nil {
		Cancel(comp, fmt.Sprintf("アイテム転送エラー: %s", err.Error()))
		return err
	}

	Complete(comp)
	return nil
}

// Finish はアイテム転送完了時の処理を実行する
func (tb *TransferBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム転送アクティビティ完了", "actor", actor)
	return nil
}

// Canceled はアイテム転送キャンセル時の処理を実行する
func (tb *TransferBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム転送キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performTransfer はアイテムを受取人のバックパックに移動する
func (tb *TransferBehavior) performTransfer(comp *gc.Activity, world w.World) error {
	p, ok := comp.Params.(*gc.TransferParams)
	if !ok {
		return fmt.Errorf("転送対象が指定されていません")
	}
	item := p.Target
	recipient := p.Recipient

	// 渡す主体はアクターでなくアイテムの現所有者にする。補給ではアクターの隊員がリーダーの
	// プールから引くため、アクターを主体にすると「隊員は隊員に渡した」と自己転送の誤ログになる。
	giver := world.Components.LocationInBackpack.Get(item).Owner
	giverName := query.GetEntityName(giver, world)
	recipientName := query.GetEntityName(recipient, world)

	// 実際に渡す個数を確定する。Count が0以下、または在庫以上なら在庫すべてを渡す。
	moving := query.GetEntityCount(world, item)
	if p.Count > 0 && p.Count < moving {
		moving = p.Count
	}

	// ログ名は転送前に確定させる。在庫全体でなく実際に移す個数で表示する。
	// query.FormatItemName は在庫数を出すので分割転送には使えない。
	itemName := "Unknown Item"
	if nameComp := world.Components.Name.Get(item); nameComp != nil {
		itemName = nameComp.Name
	}
	if moving > 1 {
		itemName = fmt.Sprintf("%s(%d個)", itemName, moving)
	}

	if err := lifecycle.TransferUnits(world, item, recipient, p.Count); err != nil {
		return fmt.Errorf("アイテム転送に失敗: %w", err)
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
