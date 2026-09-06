package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 代謝の定数。値は実プレイで調整する
const (
	// metabolismVitBonus は VIT 1 あたり回復へ足す%
	metabolismVitBonus = 3
	// metabolismSleepingBonus は睡眠中に回復へ足す基準%。寝具 Quality を掛ける
	metabolismSleepingBonus = 50
)

// fatigueRecoveryPenalty は疲労段階が回復レートから引く%を返す。速度・命中の不調経由とは別の、
// 回復専用のゲージ直読み。値は実プレイで調整する
func fatigueRecoveryPenalty(level gc.FatigueLevel) int {
	switch level {
	case gc.FatigueRested, gc.FatigueNormal:
		return 0
	case gc.FatigueTired:
		return 10
	case gc.FatigueExhausted:
		return 25
	}
	return 0
}

// hungerRecoveryPenalty は空腹段階が回復レートから引く%を返す
func hungerRecoveryPenalty(level gc.HungerLevel) int {
	switch level {
	case gc.HungerSatiated, gc.HungerNormal:
		return 0
	case gc.HungerHungry:
		return 10
	case gc.HungerStarving:
		return 20
	}
	return 0
}

// RecoverySources は回復レートへの寄与を内訳として返す。VIT・睡眠が上げ、疲労・空腹が下げる。
// 疲労・空腹の効き目はその意識低下量そのもので、速度・命中と同じ1つの値を読む。
// 怪我・病気は意識を下げるが回復には効かせない。自分の不調が自分の回復を止める悪循環を避けるため。
// Metabolism と Effects タブが同じこの導出を読むので値と内訳がずれない
func RecoverySources(world w.World, entity ecs.Entity) []gc.ProficiencySource {
	var srcs []gc.ProficiencySource

	// 疲労・空腹はゲージを直読みして回復を下げる。速度・命中は不調経由だが、回復だけは不調・意識を
	// 経由しない。怪我・病気が自分の回復を止める悪循環と、整数切り捨てで治癒が止まるのを避けるため
	if world.Components.Fatigue.Has(entity) {
		f := world.Components.Fatigue.Get(entity)
		if v := fatigueRecoveryPenalty(f.GetLevel()); v != 0 {
			srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceFatigue, Fatigue: f.GetLevel(), Value: -v})
		}
	}
	if world.Components.Hunger.Has(entity) {
		level := world.Components.Hunger.Get(entity).GetLevel()
		if v := hungerRecoveryPenalty(level); v != 0 {
			srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceHunger, Hunger: level, Value: -v})
		}
	}

	if world.Components.Abilities.Has(entity) {
		vit := world.Components.Abilities.Get(entity).Vitality.Total
		if v := vit * metabolismVitBonus; v != 0 {
			srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceAbility, Ability: gc.AblVIT, Amount: vit, Value: v})
		}
	}
	if world.Components.Sleeping.Has(entity) {
		quality := world.Components.Sleeping.Get(entity).Quality
		if v := quality.ApplyInt(metabolismSleepingBonus); v != 0 {
			srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceSleeping, Value: v})
		}
	}
	return srcs
}

// Metabolism は HP の自然回復と病気の回復にかかる速度係数を返す。基準は 100、下限は 0。
// 実効意識・VIT・睡眠から導く。疲労・空腹は意識を通じて効く。Effects タブの内訳と同じ導出を読む
func Metabolism(world w.World, entity ecs.Entity) consts.Percent {
	total := int(consts.PercentBase)
	for _, s := range RecoverySources(world, entity) {
		total += s.Value
	}
	return consts.Percent(max(total, 0))
}
