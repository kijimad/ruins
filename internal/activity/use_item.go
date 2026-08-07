package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// UseItemBehavior はBehaviorの実装
type UseItemBehavior struct{}

// Info はBehaviorの実装
func (u *UseItemBehavior) Info() Info {
	return Info{
		Name:            "Use Item",
		Description:     "Use an item",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (u *UseItemBehavior) Name() gc.BehaviorName {
	return gc.BehaviorUseItem
}

// NewUseItemActivity は使用アイテムを指定してアイテム使用アクティビティを組む。
func NewUseItemActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorUseItem, 0)
	comp.Params = &gc.UseItemParams{Target: target}
	return comp
}

// Validate はBehaviorの実装
func (u *UseItemBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.UseItemParams)
	if !ok {
		return ErrItemNotSet
	}

	item := p.Target

	// 何らかの効果があるかチェック
	hasEffect := world.Components.ProvidesHealing.Has(item) ||
		world.Components.ProvidesNutrition.Has(item) ||
		world.Components.InflictsDamage.Has(item)

	if !hasEffect {
		return ErrItemNoEffect
	}

	// アクターがHPコンポーネントを持っているかチェック
	if !world.Components.HP.Has(actor) {
		return ErrActorNoHP
	}

	return nil
}

// Start はBehaviorの実装
func (u *UseItemBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	if p, ok := comp.Params.(*gc.UseItemParams); ok {
		log.Debug("item use started", "actor", actor, "item", p.Target)
	}
	return nil
}

// DoTurn はBehaviorの実装
func (u *UseItemBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.UseItemParams)
	if !ok {
		Cancel(comp, "item is not set")
		return ErrItemNotSet
	}

	item := p.Target

	// 回復効果があるかチェック
	if world.Components.ProvidesHealing.Has(item) {
		healing := world.Components.ProvidesHealing.Get(item)
		if err := u.applyHealing(comp, actor, world, healing, item); err != nil {
			Cancel(comp, fmt.Sprintf("healing processing error: %s", err.Error()))
			return err
		}
	}

	// 空腹度回復効果があるかチェック
	if world.Components.ProvidesNutrition.Has(item) {
		nutrition := world.Components.ProvidesNutrition.Get(item)
		if err := u.applyNutrition(comp, actor, world, nutrition.Amount, item); err != nil {
			Cancel(comp, fmt.Sprintf("nutrition recovery processing error: %s", err.Error()))
			return err
		}
	}

	// ダメージ効果があるかチェック
	if world.Components.InflictsDamage.Has(item) {
		damage := world.Components.InflictsDamage.Get(item)
		// 共通のダメージ処理を使用
		gameaction.ApplyDamage(world, actor, damage.Amount, actor)
	}

	// 消費可能アイテムの場合は削除または個数を減らす
	if world.Components.Consumable.Has(item) {
		if err := lifecycle.ChangeItemCount(world, item, -1); err != nil {
			return fmt.Errorf("failed to consume item: %w", err)
		}
	}

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (u *UseItemBehavior) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item use finished", "actor", actor)
	return nil
}

// Canceled はBehaviorの実装
func (u *UseItemBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("item use canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// applyHealing は回復処理を適用する
func (u *UseItemBehavior) applyHealing(_ *gc.Activity, actor ecs.Entity, world w.World, healing *gc.ProvidesHealing, item ecs.Entity) error {
	// 最大HPを基準に実際の回復量を計算する（絶対量指定の場合は基準値は無視される）
	hp := world.Components.HP.Get(actor)
	amount := healing.Calc(hp.Max)

	// 回復効果倍率を適用する
	if world.Components.CharModifiers.Has(actor) {
		mods := world.Components.CharModifiers.Get(actor)
		amount = mods.HealingEffect.ApplyInt(amount)
	}
	if amount < 1 {
		amount = 1
	}

	actualHealing := gameaction.ApplyHealing(world, actor, amount)

	u.logItemUse(actor, world, item, actualHealing, true)

	return nil
}

// applyNutrition は空腹度回復処理を適用する
func (u *UseItemBehavior) applyNutrition(_ *gc.Activity, actor ecs.Entity, world w.World, amount int, item ecs.Entity) error {
	if !world.Components.Hunger.Has(actor) {
		return nil
	}
	hunger := world.Components.Hunger.Get(actor)

	// 満腹度を増加させる（値が大きいほど満腹）
	hunger.Increase(amount)

	// 満腹状態になったかチェック
	isSatiated := hunger.GetLevel() == gc.HungerSatiated

	u.logNutritionUse(actor, world, item, isSatiated)

	return nil
}

// logItemUse はアイテム使用のログを出力する
func (u *UseItemBehavior) logItemUse(actor ecs.Entity, world w.World, item ecs.Entity, amount int, isHealing bool) {
	// プレイヤーが関わる場合のみログ出力
	if !world.Components.Player.Has(actor) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := query.GetEntityName(actor, world)

	logger := gamelog.New(query.GetGameLog(world))
	logger.Fmt(query.T(world, "%s used %s"), query.NameSegment(actor, actorName, world), gamelog.Item(itemName))

	if isHealing {
		logger.Append(query.T(world, " recovered %d HP.", amount))
	} else {
		logger.Append(query.T(world, " took %d damage.", amount))
	}

	logger.Log()
}

// logNutritionUse は空腹度回復のログを出力する
func (u *UseItemBehavior) logNutritionUse(actor ecs.Entity, world w.World, item ecs.Entity, isSatiated bool) {
	// プレイヤーが関わる場合のみログ出力
	if !world.Components.Player.Has(actor) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := query.GetEntityName(actor, world)

	logger := gamelog.New(query.GetGameLog(world))
	logger.Fmt(query.T(world, "%s ate %s"), query.NameSegment(actor, actorName, world), gamelog.Item(itemName))

	if isSatiated {
		logger.Append(query.T(world, "Full."))
	}

	logger.Log()
}

// getItemName はアイテムの名前を取得する
func (u *UseItemBehavior) getItemName(item ecs.Entity, world w.World) string {
	if world.Components.Name.Has(item) {
		return world.Components.Name.Get(item).Name
	}
	return query.T(world, "Item")
}
