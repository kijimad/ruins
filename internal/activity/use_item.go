package activity

import (
	"fmt"
	"slices"

	"github.com/kijimaD/ruins/internal/i18n"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/oapi"
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

	// Use は効果のあるアイテムにしか提示されない。ここで効果なしなのは不変条件違反
	hasEffect := false
	for _, e := range useEffects {
		if e.present(world, item) {
			hasEffect = true
			break
		}
	}
	if !hasEffect {
		return fmt.Errorf("item has no effect")
	}

	if !world.Components.HP.Has(actor) {
		return fmt.Errorf("actor has no HP component")
	}

	// 各効果の使用前検証。空振りなど使えない理由があれば使う前に弾く
	for _, e := range useEffects {
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
		Cancel(comp, "item is not set")
		return ErrParamsTypeMismatch
	}

	item := p.Target

	// アイテムが持つ効果を順に適用する。効果の一覧は useEffects が単一の真実
	for _, e := range useEffects {
		if !e.present(world, item) {
			continue
		}
		if err := e.apply(u, comp, actor, item, world); err != nil {
			Cancel(comp, i18n.Msgid(fmt.Sprintf("use effect error: %s", err.Error())))
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
	amount = max(query.ProficiencyValue(world, actor, oapi.ProficiencyIdHealingEffect).ApplyInt(amount), 1)

	actualHealing := gameaction.ApplyHealing(world, actor, amount)

	u.logItemUse(actor, world, item, actualHealing, true)

	return nil
}

// mostSevereTreatable は Treats に一致する不調を全部位から探し、最も重い1つを返す。無ければ nil。
// Validate の使用可否判定と applyRemedy の適用が同じ探索を共有する
func mostSevereTreatable(hs *gc.HealthStatus, remedy *gc.Remedy) *gc.HealthCondition {
	var best *gc.HealthCondition
	for i := range hs.Parts {
		for j := range hs.Parts[i].Conditions {
			c := &hs.Parts[i].Conditions[j]
			if slices.Contains(remedy.Treats, c.Type) && (best == nil || c.Severity > best.Severity) {
				best = c
			}
		}
	}
	return best
}

// applyRemedy は治療を適用する。最も重い一致不調を治療済みにする。
// 治療は即座には治さず、TendQuality を立てて回復軌道へ乗せるだけ。実際の回復は ConditionSystem が進める。
// 治療した不調があれば true を返す。一致する不調が無ければ何もせず false を返す
func (u *UseItemBehavior) applyRemedy(actor ecs.Entity, world w.World, remedy *gc.Remedy, item ecs.Entity) bool {
	if !world.Components.HealthStatus.Has(actor) {
		return false
	}
	best := mostSevereTreatable(world.Components.HealthStatus.Get(actor), remedy)
	if best == nil {
		return false
	}
	best.TendQuality = remedy.Potency
	u.logRemedy(actor, world, item, best.Type)
	return true
}

// isRemedyOnly はアイテムが治療だけを効果に持つかを返す。治療専用が空振りするなら Validate で使用を弾く。
// 治療以外の効果を1つでも持てば専用でない。useEffects を見るので効果を足しても追従する
func (u *UseItemBehavior) isRemedyOnly(world w.World, item ecs.Entity) bool {
	if !world.Components.Remedy.Has(item) {
		return false
	}
	for _, e := range useEffects {
		if _, isRemedy := e.(remedyEffect); isRemedy {
			continue
		}
		if e.present(world, item) {
			return false
		}
	}
	return true
}

// logRemedy は治療したことをログに出す。プレイヤーが使ったときだけ出す
func (u *UseItemBehavior) logRemedy(actor ecs.Entity, world w.World, item ecs.Entity, treated gc.ConditionType) {
	if !world.Components.Player.Has(actor) {
		return
	}
	actorMarkup := query.NameMarkup(actor, query.GetEntityName(actor, world), world)
	itemMarkup := gamelog.Tag("item", u.getItemName(item, world))
	condName := query.ConditionName(world, treated)
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s used %s and treated %s.", actorMarkup, itemMarkup, condName)).
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
		stage = oapi.Fresh
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
		gameaction.ContractIllness(world, actor, world.Resources.ConditionDefs.ByOnset(oapi.RottenFood))
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
	if isHealing {
		logger.Markup(query.T(world, "%s used %s and recovered %d HP.", actorMarkup, itemMarkup, amount))
	} else {
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
