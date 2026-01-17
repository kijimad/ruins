package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/gamelog"
	w "github.com/kijimaD/ruins/lib/world"
	"github.com/kijimaD/ruins/lib/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// UseItemActivity はActivityInterfaceの実装
type UseItemActivity struct{}

// Info はActivityInterfaceの実装
func (u *UseItemActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "アイテム使用",
		Description:     "アイテムを使う",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 100,
		TotalRequiredAP: 100,
	}
}

// String はActivityInterfaceの実装
func (u *UseItemActivity) String() string {
	return "UseItem"
}

// Validate はActivityInterfaceの実装
func (u *UseItemActivity) Validate(act *Activity, world w.World) error {
	if act.Target == nil {
		return ErrItemNotSet
	}

	item := *act.Target

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
	if !act.Actor.HasComponent(world.Components.Pools) {
		return ErrActorNoPools
	}

	return nil
}

// Start はActivityInterfaceの実装
func (u *UseItemActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテム使用開始", "actor", act.Actor, "item", *act.Target)
	return nil
}

// DoTurn はActivityInterfaceの実装
func (u *UseItemActivity) DoTurn(act *Activity, world w.World) error {
	if act.Target == nil {
		act.Cancel("アイテムが指定されていません")
		return ErrItemNotSet
	}

	item := *act.Target

	// 回復効果があるかチェック
	if healing := world.Components.ProvidesHealing.Get(item); healing != nil {
		healingComponent := healing.(*gc.ProvidesHealing)
		if err := u.applyHealing(act, world, healingComponent.Amount, item); err != nil {
			act.Cancel(fmt.Sprintf("回復処理エラー: %s", err.Error()))
			return err
		}
	}

	// 空腹度回復効果があるかチェック
	if nutrition := world.Components.ProvidesNutrition.Get(item); nutrition != nil {
		nutritionComponent := nutrition.(*gc.ProvidesNutrition)
		if err := u.applyNutrition(act, world, nutritionComponent.Amount, item); err != nil {
			act.Cancel(fmt.Sprintf("空腹度回復処理エラー: %s", err.Error()))
			return err
		}
	}

	// ダメージ効果があるかチェック
	if damage := world.Components.InflictsDamage.Get(item); damage != nil {
		damageComponent := damage.(*gc.InflictsDamage)
		// 共通のダメージ処理を使用
		worldhelper.ApplyDamage(world, act.Actor, damageComponent.Amount, act.Actor)
	}

	// 消費可能アイテムの場合は削除または個数を減らす
	if item.HasComponent(world.Components.Consumable) {
		if err := worldhelper.ChangeItemCount(world, item, -1); err != nil {
			return fmt.Errorf("アイテムの消費に失敗: %w", err)
		}
	}

	act.Complete()
	return nil
}

// Finish はActivityInterfaceの実装
func (u *UseItemActivity) Finish(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテム使用完了", "actor", act.Actor)
	return nil
}

// Canceled はActivityInterfaceの実装
func (u *UseItemActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("アイテム使用キャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}

// applyHealing は回復処理を適用する
func (u *UseItemActivity) applyHealing(act *Activity, world w.World, amounter gc.Amounter, item ecs.Entity) error {
	// Amounterから実際の回復量を取得
	var amount int
	switch amt := amounter.(type) {
	case gc.NumeralAmount:
		amount = amt.Calc()
	case gc.RatioAmount:
		// 最大HPに対する割合で回復
		pools := world.Components.Pools.Get(act.Actor).(*gc.Pools)
		amount = amt.Calc(pools.HP.Max)
	default:
		return fmt.Errorf("未対応のAmounterタイプ: %T", amounter)
	}

	// 共通の回復処理を使用
	actualHealing := worldhelper.ApplyHealing(world, act.Actor, amount)

	u.logItemUse(act, world, item, actualHealing, true)

	return nil
}

// applyNutrition は空腹度回復処理を適用する
func (u *UseItemActivity) applyNutrition(act *Activity, world w.World, amount int, item ecs.Entity) error {
	hungerComp := world.Components.Hunger.Get(act.Actor)
	if hungerComp == nil {
		return nil
	}

	hunger := hungerComp.(*gc.Hunger)

	// 満腹度を増加させる（値が大きいほど満腹）
	hunger.Increase(amount)

	// 満腹状態になったかチェック
	isSatiated := hunger.GetLevel() == gc.HungerSatiated

	u.logNutritionUse(act, world, item, isSatiated)

	return nil
}

// logItemUse はアイテム使用のログを出力する
func (u *UseItemActivity) logItemUse(act *Activity, world w.World, item ecs.Entity, amount int, isHealing bool) {
	// プレイヤーが関わる場合のみログ出力
	if !isPlayerActivity(act, world) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := worldhelper.GetEntityName(act.Actor, world)

	logger := gamelog.New(gamelog.FieldLog)
	logger.Build(func(l *gamelog.Logger) {
		worldhelper.AppendNameWithColor(l, act.Actor, actorName, world)
	}).Append(" は ").ItemName(itemName).Append(" を使った。")

	if isHealing {
		logger.Append(fmt.Sprintf(" HPが %d 回復した。", amount))
	} else {
		logger.Append(fmt.Sprintf(" %d のダメージを受けた。", amount))
	}

	logger.Log()
}

// logNutritionUse は空腹度回復のログを出力する
func (u *UseItemActivity) logNutritionUse(act *Activity, world w.World, item ecs.Entity, isSatiated bool) {
	// プレイヤーが関わる場合のみログ出力
	if !isPlayerActivity(act, world) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := worldhelper.GetEntityName(act.Actor, world)

	logger := gamelog.New(gamelog.FieldLog)
	logger.Build(func(l *gamelog.Logger) {
		worldhelper.AppendNameWithColor(l, act.Actor, actorName, world)
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
