package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// UseItemActivity はBehaviorの実装
type UseItemActivity struct{}

// Info はBehaviorの実装
func (u *UseItemActivity) Info() Info {
	return Info{
		Name:            "アイテム使用",
		Description:     "アイテムを使う",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (u *UseItemActivity) Name() gc.BehaviorName {
	return gc.BehaviorUseItem
}

// Validate はBehaviorの実装
func (u *UseItemActivity) Validate(comp *gc.CurrentActivity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return ErrItemNotSet
	}

	item := *comp.Target

	// アイテムエンティティにItemコンポーネントがあるかチェック
	if !item.HasComponent(world.Components.Item) {
		return ErrInvalidItem
	}

	// 何らかの効果があるかチェック
	hasEffect := item.HasComponent(world.Components.ProvidesHealing) ||
		item.HasComponent(world.Components.ProvidesNutrition) ||
		item.HasComponent(world.Components.InflictsDamage)

	if !hasEffect {
		return ErrItemNoEffect
	}

	// アクターがPoolsコンポーネントを持っているかチェック
	if !actor.HasComponent(world.Components.Pools) {
		return ErrActorNoPools
	}

	return nil
}

// Start はBehaviorの実装
func (u *UseItemActivity) Start(comp *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム使用開始", "actor", actor, "item", *comp.Target)
	return nil
}

// DoTurn はBehaviorの実装
func (u *UseItemActivity) DoTurn(comp *gc.CurrentActivity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		Cancel(comp, "アイテムが指定されていません")
		return ErrItemNotSet
	}

	item := *comp.Target

	// 回復効果があるかチェック
	if healing := world.Components.ProvidesHealing.Get(item); healing != nil {
		healingComponent := healing.(*gc.ProvidesHealing)
		if err := u.applyHealing(comp, actor, world, healingComponent.Amount, item); err != nil {
			Cancel(comp, fmt.Sprintf("回復処理エラー: %s", err.Error()))
			return err
		}
	}

	// 空腹度回復効果があるかチェック
	if nutrition := world.Components.ProvidesNutrition.Get(item); nutrition != nil {
		nutritionComponent := nutrition.(*gc.ProvidesNutrition)
		if err := u.applyNutrition(comp, actor, world, nutritionComponent.Amount, item); err != nil {
			Cancel(comp, fmt.Sprintf("空腹度回復処理エラー: %s", err.Error()))
			return err
		}
	}

	// ダメージ効果があるかチェック
	if damage := world.Components.InflictsDamage.Get(item); damage != nil {
		damageComponent := damage.(*gc.InflictsDamage)
		// 共通のダメージ処理を使用
		worldhelper.ApplyDamage(world, actor, damageComponent.Amount, actor)
	}

	// 消費可能アイテムの場合は削除または個数を減らす
	if item.HasComponent(world.Components.Consumable) {
		if err := worldhelper.ChangeItemCount(world, item, -1); err != nil {
			return fmt.Errorf("アイテムの消費に失敗: %w", err)
		}
	}

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (u *UseItemActivity) Finish(_ *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム使用完了", "actor", actor)
	return nil
}

// Canceled はBehaviorの実装
func (u *UseItemActivity) Canceled(comp *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム使用キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// applyHealing は回復処理を適用する
func (u *UseItemActivity) applyHealing(_ *gc.CurrentActivity, actor ecs.Entity, world w.World, amounter gc.Amounter, item ecs.Entity) error {
	// Amounterから実際の回復量を取得
	var amount int
	switch amt := amounter.(type) {
	case gc.NumeralAmount:
		amount = amt.Calc()
	case gc.RatioAmount:
		// 最大HPに対する割合で回復
		pools := world.Components.Pools.Get(actor).(*gc.Pools)
		amount = amt.Calc(pools.HP.Max)
	default:
		return fmt.Errorf("未対応のAmounterタイプ: %T", amounter)
	}

	// 共通の回復処理を使用
	actualHealing := worldhelper.ApplyHealing(world, actor, amount)

	u.logItemUse(actor, world, item, actualHealing, true)

	return nil
}

// applyNutrition は空腹度回復処理を適用する
func (u *UseItemActivity) applyNutrition(_ *gc.CurrentActivity, actor ecs.Entity, world w.World, amount int, item ecs.Entity) error {
	hungerComp := world.Components.Hunger.Get(actor)
	if hungerComp == nil {
		return nil
	}

	hunger := hungerComp.(*gc.Hunger)

	// 満腹度を増加させる（値が大きいほど満腹）
	hunger.Increase(amount)

	// 満腹状態になったかチェック
	isSatiated := hunger.GetLevel() == gc.HungerSatiated

	u.logNutritionUse(actor, world, item, isSatiated)

	return nil
}

// logItemUse はアイテム使用のログを出力する
func (u *UseItemActivity) logItemUse(actor ecs.Entity, world w.World, item ecs.Entity, amount int, isHealing bool) {
	// プレイヤーが関わる場合のみログ出力
	if !actor.HasComponent(world.Components.Player) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := worldhelper.GetEntityName(actor, world)

	logger := gamelog.New(gamelog.FieldLog)
	logger.Build(func(l *gamelog.Logger) {
		worldhelper.AppendNameWithColor(l, actor, actorName, world)
	}).Append(" は ").ItemName(itemName).Append(" を使った。")

	if isHealing {
		logger.Append(fmt.Sprintf(" HPが %d 回復した。", amount))
	} else {
		logger.Append(fmt.Sprintf(" %d のダメージを受けた。", amount))
	}

	logger.Log()
}

// logNutritionUse は空腹度回復のログを出力する
func (u *UseItemActivity) logNutritionUse(actor ecs.Entity, world w.World, item ecs.Entity, isSatiated bool) {
	// プレイヤーが関わる場合のみログ出力
	if !actor.HasComponent(world.Components.Player) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := worldhelper.GetEntityName(actor, world)

	logger := gamelog.New(gamelog.FieldLog)
	logger.Build(func(l *gamelog.Logger) {
		worldhelper.AppendNameWithColor(l, actor, actorName, world)
	}).Append(" は ").ItemName(itemName).Append(" を食べた。")

	if isSatiated {
		logger.Append("満腹だ。")
	}

	logger.Log()
}

// getItemName はアイテムの名前を取得する
func (u *UseItemActivity) getItemName(item ecs.Entity, world w.World) string {
	name := world.Components.Name.Get(item)
	if name != nil {
		return name.(*gc.Name).Name
	}
	return "アイテム"
}
