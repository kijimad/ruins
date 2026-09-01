package activity

import (
	"fmt"
	"slices"

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
		return ErrParamsTypeMismatch
	}

	item := p.Target

	// 何らかの効果があるかチェック
	hasEffect := world.Components.ProvidesHealing.Has(item) ||
		world.Components.ProvidesNutrition.Has(item) ||
		world.Components.InflictsDamage.Has(item) ||
		world.Components.Remedy.Has(item)

	// Use は効果のあるアイテムにしか提示されない。ここで効果なしなのは不変条件違反
	if !hasEffect {
		return fmt.Errorf("item has no effect")
	}

	if !world.Components.HP.Has(actor) {
		return fmt.Errorf("actor has no HP component")
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
		return ErrParamsTypeMismatch
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

	// 治療効果があるかチェック。治療専用のアイテムが何も治療しなければ消費しない
	remedyWasted := false
	if world.Components.Remedy.Has(item) {
		remedy := world.Components.Remedy.Get(item)
		treated := u.applyRemedy(actor, world, remedy, item)
		remedyWasted = !treated && u.isRemedyOnly(world, item)
	}

	// 消費可能アイテムの場合は削除または個数を減らす。無駄になった治療は消費しない
	if world.Components.Consumable.Has(item) && !remedyWasted {
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

// applyRemedy は治療を適用する。Treats に一致する不調を全部位から集め、最も重い1つを治療済みにする。
// 治療は即座には治さず、TendQuality を立てて回復軌道へ乗せるだけ。実際の回復は ConditionSystem が進める。
// 治療した不調があれば true を返す。一致する不調が無ければ何もせず false を返す
func (u *UseItemBehavior) applyRemedy(actor ecs.Entity, world w.World, remedy *gc.Remedy, item ecs.Entity) bool {
	if !world.Components.HealthStatus.Has(actor) {
		return false
	}
	hs := world.Components.HealthStatus.Get(actor)

	var best *gc.HealthCondition
	for i := range hs.Parts {
		for j := range hs.Parts[i].Conditions {
			c := &hs.Parts[i].Conditions[j]
			if slices.Contains(remedy.Treats, c.Type) && (best == nil || c.Severity > best.Severity) {
				best = c
			}
		}
	}
	if best == nil {
		return false
	}
	best.TendQuality = remedy.Potency
	u.logRemedy(actor, world, item, best.Type)
	return true
}

// isRemedyOnly はアイテムが治療だけを効果に持つかを返す。治療が空振りしたとき消費するかの判定に使う
func (u *UseItemBehavior) isRemedyOnly(world w.World, item ecs.Entity) bool {
	return world.Components.Remedy.Has(item) &&
		!world.Components.ProvidesHealing.Has(item) &&
		!world.Components.ProvidesNutrition.Has(item) &&
		!world.Components.InflictsDamage.Has(item)
}

// logRemedy は治療したことをログに出す。プレイヤーが使ったときだけ出す
func (u *UseItemBehavior) logRemedy(actor ecs.Entity, world w.World, item ecs.Entity, treated gc.ConditionType) {
	if !world.Components.Player.Has(actor) {
		return
	}
	actorMarkup := query.NameMarkup(actor, query.GetEntityName(actor, world), world)
	itemMarkup := gamelog.Tag("item", u.getItemName(item, world))
	condName := query.T(world, gc.ConditionTypeDisplayName(treated))
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s used %s and treated %s.", actorMarkup, itemMarkup, condName)).
		Log()
}

// rottenNutritionPercent は腐敗した食料から得られる栄養の割合。満額の3割。
const rottenNutritionPercent = 30

// rottenIllnessChancePercent は腐敗食を食べたとき病気を発症する確率。値は実プレイで調整する
const rottenIllnessChancePercent = 25

// applyNutrition は空腹度回復処理を適用する。鮮度に応じて栄養を調整する。
func (u *UseItemBehavior) applyNutrition(_ *gc.Activity, actor ecs.Entity, world w.World, amount int, item ecs.Entity) error {
	if !world.Components.Hunger.Has(actor) {
		return nil
	}
	hunger := world.Components.Hunger.Get(actor)

	// Perishable を持たない食料は新鮮扱いで満額
	stage, ok := query.FreshnessStageOf(world, item)
	if !ok {
		stage = gc.FreshnessFresh
	}

	var nutrition int
	switch stage {
	case gc.FreshnessFresh:
		nutrition = amount
	case gc.FreshnessStale:
		nutrition = amount / 2
	case gc.FreshnessRotten:
		nutrition = amount * rottenNutritionPercent / 100
	}
	// 鮮度で減った栄養が整数除算で0に落ちても、腐敗食は非常食として最低1は与える。
	// balance で小さい栄養値を設定したとき、劣化・腐敗が完全にゼロ効果になる罠を防ぐ
	if amount > 0 && nutrition < 1 {
		nutrition = 1
	}
	hunger.Increase(nutrition)

	// 腐敗食は低確率で病気を発症させる。放置すると悪化するので早めの治療を促す
	if stage == gc.FreshnessRotten && world.Resources.Config.RNG.IntN(100) < rottenIllnessChancePercent {
		gameaction.ContractIllness(world, actor, gc.ConditionLiverIllness)
	}

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

	actorMarkup := query.NameMarkup(actor, actorName, world)
	itemMarkup := gamelog.Tag("item", itemName)
	logger := gamelog.New(query.GetGameLog(world))
	if isHealing {
		logger.Markup(query.T(world, "%s used %s and recovered %d HP.", actorMarkup, itemMarkup, amount))
	} else {
		logger.Markup(query.T(world, "%s used %s and took %d damage.", actorMarkup, itemMarkup, amount))
	}
	logger.Log()
}

// logNutritionUse は空腹度回復のログを出力する。腐敗する食料は鮮度を添える
func (u *UseItemBehavior) logNutritionUse(actor ecs.Entity, world w.World, item ecs.Entity, isSatiated bool) {
	// プレイヤーが関わる場合のみログ出力
	if !world.Components.Player.Has(actor) {
		return
	}

	itemName := u.getItemName(item, world)
	actorName := query.GetEntityName(actor, world)

	actorMarkup := query.NameMarkup(actor, actorName, world)
	itemMarkup := gamelog.Tag("item", itemName)
	logger := gamelog.New(query.GetGameLog(world))

	// 腐敗する食料は鮮度を名前に添える。腐敗しない食料はそのまま
	if stage, ok := query.FreshnessStageOf(world, item); !ok {
		logger.Markup(query.T(world, "%s ate %s", actorMarkup, itemMarkup))
	} else {
		switch stage {
		case gc.FreshnessFresh:
			logger.Markup(query.T(world, "%s ate fresh %s.", actorMarkup, itemMarkup))
		case gc.FreshnessStale:
			logger.Markup(query.T(world, "%s ate old %s.", actorMarkup, itemMarkup))
		case gc.FreshnessRotten:
			logger.Markup(query.T(world, "%s ate spoiled %s.", actorMarkup, itemMarkup))
		}
	}
	logger.Log()

	// 満腹になったら別行で知らせる
	if isSatiated {
		gamelog.New(query.GetGameLog(world)).Markup(query.T(world, "%s is now full.", actorMarkup)).Log()
	}
}

// getItemName はアイテムの名前を取得する
func (u *UseItemBehavior) getItemName(item ecs.Entity, world w.World) string {
	if world.Components.Name.Has(item) {
		return query.T(world, world.Components.Name.Get(item).Name)
	}
	return query.T(world, "Item")
}
