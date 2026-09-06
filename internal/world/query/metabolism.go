package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 代謝の定数。値は実プレイで調整する。VIT の寄与は components の spec 表が持つ
const (
	// metabolismSatiatedBonus は満腹のとき代謝倍率へ足す%
	metabolismSatiatedBonus = 20
	// metabolismHungryPenalty は空腹のとき代謝倍率から引く%
	metabolismHungryPenalty = 30
	// metabolismStarvingPenalty は飢餓のとき代謝倍率から引く%
	metabolismStarvingPenalty = 60
	// metabolismSleepingBonus は睡眠中に代謝倍率へ足す基準%。寝具 Quality を掛ける
	metabolismSleepingBonus = 50
)

// HungerRecoveryDelta は空腹段階が代謝倍率へ与える加算%を返す。満腹は+、空腹・飢餓は-、普通は0。
// Metabolism と Basic タブの内訳表示が同じ値を見るための単一の算出点。
func HungerRecoveryDelta(level gc.HungerLevel) consts.Percent {
	switch level {
	case gc.HungerSatiated:
		return metabolismSatiatedBonus
	case gc.HungerNormal:
		return 0
	case gc.HungerHungry:
		return -metabolismHungryPenalty
	case gc.HungerStarving:
		return -metabolismStarvingPenalty
	}
	return 0
}

// recoverySources は ModRecovery への状態由来の寄与を内訳として返す。空腹・疲労・睡眠が加算で効く。
// VIT は spec の能力ソースとして forEachModifierSource が出すのでここには含めない。
// Metabolism と Effects タブの内訳がこの1箇所を読むので、値と内訳がずれない。
func recoverySources(world w.World, entity ecs.Entity) []gc.ModifierSource {
	var srcs []gc.ModifierSource

	if world.Components.Hunger.Has(entity) {
		level := world.Components.Hunger.Get(entity).GetLevel()
		if v := int(HungerRecoveryDelta(level)); v != 0 {
			srcs = append(srcs, gc.ModifierSource{Kind: gc.SourceHunger, Hunger: level, Value: v})
		}
	}
	if world.Components.Fatigue.Has(entity) {
		f := world.Components.Fatigue.Get(entity)
		if v := int(f.Penalty().RecoveryAdd); v != 0 {
			srcs = append(srcs, gc.ModifierSource{Kind: gc.SourceFatigue, Fatigue: f.GetLevel(), Value: v})
		}
	}
	if world.Components.Sleeping.Has(entity) {
		quality := world.Components.Sleeping.Get(entity).Quality
		if v := quality.ApplyInt(metabolismSleepingBonus); v != 0 {
			srcs = append(srcs, gc.ModifierSource{Kind: gc.SourceSleeping, Value: v})
		}
	}
	return srcs
}

// Metabolism は HP の自然回復と病気の回復にかかる速度係数を返す。基準は 100、下限は 0。
// ModRecovery の効果値そのもので、Effects タブの内訳と同じ導出を読む
func Metabolism(world w.World, entity ecs.Entity) consts.Percent {
	pct := max(ModifierValue(world, entity, gc.ModRecovery), 0)
	return pct
}
