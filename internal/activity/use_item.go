package activity

import (
	"fmt"
	"slices"

	"github.com/kijimaD/ruins/internal/i18n"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// UseItemBehavior はBehaviorの実装。eat が真なら食べるの効果を、偽なら使うの効果を適用する
type UseItemBehavior struct {
	eat bool
}

// effects はこの行動で適用する効果の組を返す
func (u *UseItemBehavior) effects() []useEffect {
	if u.eat {
		return eatEffects
	}
	return useEffects
}

// Info はBehaviorの実装
func (u *UseItemBehavior) Info() Info {
	name, desc := "Use Item", "Use an item"
	if u.eat {
		name, desc = "Eat", "Eat an item"
	}
	return Info{
		Name:            name,
		Description:     desc,
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (u *UseItemBehavior) Name() gc.BehaviorName {
	if u.eat {
		return gc.BehaviorEat
	}
	return gc.BehaviorUseItem
}

// NewUseItemActivity は使用アイテムを指定してアイテム使用アクティビティを組む。
func NewUseItemActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorUseItem, 0)
	comp.Params = &gc.UseItemParams{Target: target}
	return comp
}

// NewEatActivity は食べるアイテムを指定して食べるアクティビティを組む。
func NewEatActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorEat, 0)
	comp.Params = &gc.UseItemParams{Target: target}
	return comp
}

// CanUse は物が使うの効果を持ち、使えるかを返す。効果を持つ物が消費物であることは raw のスキーマが保証する
func CanUse(world w.World, item ecs.Entity) bool {
	return hasEffect(useEffects, world, item)
}

// CanEat は物が食べるの効果を持ち、食べられるかを返す
func CanEat(world w.World, item ecs.Entity) bool {
	return hasEffect(eatEffects, world, item)
}

func hasEffect(effects []useEffect, world w.World, item ecs.Entity) bool {
	return slices.ContainsFunc(effects, func(e useEffect) bool { return e.present(world, item) })
}

// Validate はBehaviorの実装
func (u *UseItemBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.UseItemParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	item := p.Target

	// 食べる・使うは、その組の効果を持つアイテムにしか提示されない。ここで効果なしなのは不変条件違反
	if !hasEffect(u.effects(), world, item) {
		return errItemNoEffect
	}

	if !world.Components.HP.Has(actor) {
		return fmt.Errorf("actor has no HP component")
	}

	// 各効果の使用前検証。空振りなど使えない理由があれば使う前に弾く
	for _, e := range u.effects() {
		if !e.present(world, item) {
			continue
		}
		if err := e.check(u, actor, item, world); err != nil {
			return err
		}
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
		return ErrParamsTypeMismatch
	}

	item := p.Target

	// アイテムが持つ効果のうち、この行動の組の効果を順に適用する
	for _, e := range u.effects() {
		if !e.present(world, item) {
			continue
		}
		if err := e.apply(u, comp, actor, item, world); err != nil {
			return err
		}
	}

	// 消費可能アイテムの場合は削除または個数を減らす。効果を適用した後に行う
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
	amount = max(query.ProficiencyValue(world, actor, rawschema.ProficiencyIdHealingEffect).ApplyInt(amount), 1)

	actualHealing := gameaction.ApplyHealing(world, actor, amount)

	u.logItemUse(actor, world, item, actualHealing, true)

	return nil
}

// remedyTarget は remedy で治療する不調を1つ選び、部位と一緒に返す。無ければ nil。
// 発症前の不調と、治療の質が remedy 以上の不調は除く。未治療を先にし、その中で重症度の段階が最も高いものを選ぶ
func remedyTarget(hs *gc.HealthStatus, remedy *gc.Remedy) (gc.BodyPart, *gc.HealthCondition) {
	var bestPart gc.BodyPart
	var best *gc.HealthCondition
	for i := range hs.Parts {
		for j := range hs.Parts[i].Conditions {
			c := &hs.Parts[i].Conditions[j]
			if !slices.Contains(remedy.Treats, c.Type) || !c.IsActive() || c.TendQuality >= remedy.Potency {
				continue
			}
			if best == nil {
				bestPart, best = gc.BodyPart(i), c
				continue
			}
			untreated, bestUntreated := c.TendQuality == 0, best.TendQuality == 0
			if (untreated && !bestUntreated) || (untreated == bestUntreated && c.Severity > best.Severity) {
				bestPart, best = gc.BodyPart(i), c
			}
		}
	}
	return bestPart, best
}

// applyRemedy は remedyTarget が選んだ不調を治療済みにする。
// 治療は即座には治さず、TendQuality を立てて回復軌道へ乗せるだけ。実際の回復は ConditionSystem が進める。
// 治療した不調があれば true を返す。選べる不調が無ければ何もせず false を返す
func (u *UseItemBehavior) applyRemedy(actor ecs.Entity, world w.World, remedy *gc.Remedy, item ecs.Entity) bool {
	if !world.Components.HealthStatus.Has(actor) {
		return false
	}
	part, best := remedyTarget(world.Components.HealthStatus.Get(actor), remedy)
	if best == nil {
		return false
	}
	best.TendQuality = remedy.Potency
	u.logRemedy(actor, world, item, part, best.Type)
	return true
}

// isRemedyOnly はこの行動の組の中で、アイテムが治療だけを効果に持つかを返す。治療専用が空振りするなら Validate で使用を弾く。
// ほかの組の効果は見ないので、栄養を持つ蜂蜜も使うの組では治療専用になる
func (u *UseItemBehavior) isRemedyOnly(world w.World, item ecs.Entity) bool {
	if !world.Components.Remedy.Has(item) {
		return false
	}
	for _, e := range u.effects() {
		if _, isRemedy := e.(remedyEffect); isRemedy {
			continue
		}
		if e.present(world, item) {
			return false
		}
	}
	return true
}

// logRemedy は治療した部位と不調をログに出す。プレイヤーが使ったときだけ出す
func (u *UseItemBehavior) logRemedy(actor ecs.Entity, world w.World, item ecs.Entity, part gc.BodyPart, treated gc.ConditionType) {
	if !world.Components.Player.Has(actor) {
		return
	}
	actorMarkup := query.NameMarkup(actor, query.GetEntityName(actor, world), world)
	itemMarkup := gamelog.Tag("item", u.getItemName(item, world))
	condName := query.ConditionName(world, treated)
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s used %s on the %s and treated %s.", actorMarkup, itemMarkup, query.BodyPartName(world, part), condName)).
		Log()
}

// applyNutrition は空腹度回復処理を適用する。鮮度に応じて栄養を調整する。
func (u *UseItemBehavior) applyNutrition(_ *gc.Activity, actor ecs.Entity, world w.World, amount int, item ecs.Entity) error {
	if !world.Components.Hunger.Has(actor) {
		return nil
	}
	hunger := world.Components.Hunger.Get(actor)

	// Perishable を持たない食料は新鮮扱いで満額
	stage, ok := query.FreshnessStageOf(world, item)
	if !ok {
		stage = rawschema.Fresh
	}

	freshness := world.Resources.NeedDefs.FreshnessOf(stage)
	nutrition := amount * freshness.NutritionPercent / 100
	// 鮮度で減った栄養が整数除算で0に落ちても、腐敗食は非常食として最低1は与える。
	// balance で小さい栄養値を設定したとき、劣化・腐敗が完全にゼロ効果になる罠を防ぐ
	if amount > 0 && nutrition < 1 {
		nutrition = 1
	}
	hunger.Increase(nutrition)

	if freshness.IllnessChancePercent > 0 && world.Resources.Config.RNG.IntN(100) < freshness.IllnessChancePercent {
		gameaction.ContractIllness(world, actor, world.Resources.ConditionDefs.ByOnset(rawschema.RottenFood))
	}

	hungerStage, _ := world.Resources.NeedDefs.HungerStageOf(hunger.GetLevel(world.Resources.NeedDefs))
	u.logNutritionUse(actor, world, item, hungerStage.ReachedLog)

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
	switch {
	case isHealing && world.Components.ProvidesNutrition.Has(item):
		// 食べたことは栄養のログが出すので、回復の量だけを出す
		logger.Markup(query.T(world, "%s recovered %d HP.", actorMarkup, amount))
	case isHealing:
		logger.Markup(query.T(world, "%s used %s and recovered %d HP.", actorMarkup, itemMarkup, amount))
	default:
		logger.Markup(query.T(world, "%s used %s and took %d damage.", actorMarkup, itemMarkup, amount))
	}
	logger.Log()
}

// logNutritionUse は空腹度回復のログを出力する。腐敗する食料は鮮度を添える。
// reachedLog は食べて達した空腹の段階のログで、空なら出さない
func (u *UseItemBehavior) logNutritionUse(actor ecs.Entity, world w.World, item ecs.Entity, reachedLog i18n.Msgid) {
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
		logger.Markup(query.T(world, "%s ate %s.", actorMarkup, itemMarkup))
	} else {
		logger.Markup(query.T(world, world.Resources.NeedDefs.FreshnessOf(stage).EatLog, actorMarkup, itemMarkup))
	}
	logger.Log()

	if reachedLog != "" {
		gamelog.New(query.GetGameLog(world)).Markup(query.T(world, reachedLog, actorMarkup)).Log()
	}
}

// getItemName はアイテムの名前を取得する
func (u *UseItemBehavior) getItemName(item ecs.Entity, world w.World) string {
	if world.Components.Name.Has(item) {
		return query.T(world, world.Components.Name.Get(item).Name)
	}
	return query.T(world, "Item")
}
