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
		Name:            "Transfer",
		Description:     "Give an item to another entity",
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
		return fmt.Errorf("transfer target is not set")
	}
	// 値型のパラメータでは未指定が無効エンティティになる。ArkのHasはゼロ値でパニックするため、
	// Aliveで存在を確かめてから所持判定へ進む
	if !world.ECS.Alive(p.Target) {
		return fmt.Errorf("transfer target is not set")
	}
	if !world.ECS.Alive(p.Recipient) {
		return fmt.Errorf("recipient is not set")
	}

	target := p.Target
	if !world.Components.LocationInBackpack.Has(target) {
		return fmt.Errorf("TransferBehavior.Validate: item is not in the backpack")
	}

	return nil
}

// Start はアイテム転送開始時の処理を実行する
func (tb *TransferBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item transfer started", "actor", actor)
	return nil
}

// DoTurn はアイテム転送アクティビティの1ターン分の処理を実行する
func (tb *TransferBehavior) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	if err := tb.performTransfer(comp, world); err != nil {
		Cancel(comp, fmt.Sprintf("item transfer error: %s", err.Error()))
		return err
	}

	Complete(comp)
	return nil
}

// Finish はアイテム転送完了時の処理を実行する
func (tb *TransferBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item transfer activity finished", "actor", actor)
	return nil
}

// Canceled はアイテム転送キャンセル時の処理を実行する
func (tb *TransferBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item transfer canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// performTransfer はアイテムを受取人のバックパックに移動する
func (tb *TransferBehavior) performTransfer(comp *gc.Activity, world w.World) error {
	p, ok := comp.Params.(*gc.TransferParams)
	if !ok {
		return fmt.Errorf("transfer target is not set")
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
	itemName := query.T(world, "Unknown Item")
	if nameComp := world.Components.Name.Get(item); nameComp != nil {
		itemName = query.T(world, nameComp.Name)
	}
	if moving > 1 {
		itemName = query.T(world, "%s (x%d)", itemName, moving)
	}

	if err := lifecycle.TransferUnits(world, item, recipient, p.Count); err != nil {
		return fmt.Errorf("failed to transfer item: %w", err)
	}

	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s handed over %s to %s.",
			query.NameMarkup(giver, giverName, world),
			gamelog.Tag("item", itemName),
			recipientName)).
		Log()

	return nil
}
