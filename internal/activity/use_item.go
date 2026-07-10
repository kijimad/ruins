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

// UseItemActivity はBehaviorの実装
type UseItemActivity struct {
	Target ecs.Entity
}

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

// BuildActivity はBehaviorの実装
func (u *UseItemActivity) BuildActivity(_ ecs.Entity, _ w.World) (*gc.Activity, error) {
	comp, err := NewActivity(u, 1)
	if err != nil {
		return nil, err
	}
	comp.Target = &u.Target
	return comp, nil
}

// Validate はBehaviorの実装
func (u *UseItemActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return ErrItemNotSet
	}

	item := *comp.Target

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
func (u *UseItemActivity) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム使用開始", "actor", actor, "item", *comp.Target)
	return nil
}

// DoTurn はBehaviorの実装
func (u *UseItemActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		Cancel(comp, "アイテムが指定されていません")
		return ErrItemNotSet
	}

	item := *comp.Target

	// 回復効果があるかチェック
	if healing := world.Components.ProvidesHealing.Get(item); healing != nil {
		healingComponent := healing
		if err := u.applyHealing(comp, actor, world, healingComponent.Amount, item); err != nil {
			Cancel(comp, fmt.Sprintf("回復処理エラー: %s", err.Error()))
			return err
		}
	}

	// 空腹度回復効果があるかチェック
	if nutrition := world.Components.ProvidesNutrition.Get(item); nutrition != nil {
		nutritionComponent := nutrition
		if err := u.applyNutrition(comp, actor, world, nutritionComponent.Amount, item); err != nil {
			Cancel(comp, fmt.Sprintf("空腹度回復処理エラー: %s", err.Error()))
			return err
		}
	}

	// ダメージ効果があるかチェック
	if damage := world.Components.InflictsDamage.Get(item); damage != nil {
		damageComponent := damage
		// 共通のダメージ処理を使用
		gameaction.ApplyDamage(world, actor, damageComponent.Amount, actor)
	}

	// 消費可能アイテムの場合は削除または個数を減らす
	if world.Components.Consumable.Has(item) {
		if err := lifecycle.ChangeItemCount(world, item, -1); err != nil {
			return fmt.Errorf("アイテムの消費に失敗: %w", err)
		}
	}

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (u *UseItemActivity) Finish(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム使用完了", "actor", actor)
	return nil
}

// Canceled はBehaviorの実装
func (u *UseItemActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("アイテム使用キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// applyHealing は回復処理を適用する
func (u *UseItemActivity) applyHealing(_ *gc.Activity, actor ecs.Entity, world w.World, amounter gc.Amounter, item ecs.Entity) error {
	// Amounterから実際の回復量を取得
	var amount int
	switch amt := amounter.(type) {
	case gc.NumeralAmount:
		amount = amt.Calc()
	case gc.RatioAmount:
		// 最大HPに対する割合で回復
		hp := world.Components.HP.Get(actor)
		amount = amt.Calc(hp.Max)
	default:
		return fmt.Errorf("未対応のAmounterタイプ: %T", amounter)
	}

	// 回復効果倍率を適用する
	if world.Components.CharModifiers.Has(actor) {
		mods := world.Components.CharModifiers.Get(actor)
		amount = amount * mods.HealingEffect / 100
	}
	if amount < 1 {
		amount = 1
	}

	actualHealing := gameaction.ApplyHealing(world, actor, amount)

	u.logItemUse(actor, world, item, actualHealing, true)

	return nil
}

// applyNutrition は空腹度回復処理を適用する
func (u *UseItemActivity) applyNutrition(_ *gc.Activity, actor ecs.Entity, world w.World, amount int, item ecs.Entity) error {
	hungerComp := world.Components.Hunger.Get(actor)
	if hungerComp == nil {
		return nil
	}

	hunger := hungerComp

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
	if !world.Components.Player.Has(actor) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := query.GetEntityName(actor, world)

	logger := gamelog.New(query.GetGameLog(world))
	logger.Build(func(l *gamelog.Logger) {
		query.AppendNameWithColor(l, actor, actorName, world)
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
	if !world.Components.Player.Has(actor) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := query.GetEntityName(actor, world)

	logger := gamelog.New(query.GetGameLog(world))
	logger.Build(func(l *gamelog.Logger) {
		query.AppendNameWithColor(l, actor, actorName, world)
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
		return name.Name
	}
	return "アイテム"
}
