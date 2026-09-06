package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// EffectiveBodyFuncs は消費側が読む最終的な身体機能を返す。全身性の低下をここ1箇所で各機能へ畳む。
// 全身性は怪我由来の痛み・全身の不調に疲労・空腹を足したもので、操作・歩行・視覚・意識すべてへ一律に効く。
// 速度・命中はこの値を読むだけでよく、意識を消費側で掛け直す必要はない。
// 部位ごとの怪我は素の HealthStatus.BodyFuncs から来る
func EffectiveBodyFuncs(world w.World, entity ecs.Entity) gc.BodyFuncs {
	raw := gc.HealthyBodyFuncs()
	if world.Components.HealthStatus.Has(entity) {
		raw = world.Components.HealthStatus.Get(entity).BodyFuncs()
	}
	// 怪我由来の全身性は 100 と素の意識の差。ここに疲労・空腹の低下を足す
	systemic := (int(consts.PercentBase) - int(raw.Consciousness)) + consciousnessPenalty(world, entity)
	sub := func(v consts.Percent) consts.Percent {
		return consts.Percent(max(int(v)-systemic, 0))
	}
	raw.Consciousness = consts.Percent(max(int(consts.PercentBase)-systemic, 0))
	raw.Manipulation = sub(raw.Manipulation)
	raw.Moving = sub(raw.Moving)
	raw.Sight = sub(raw.Sight)
	return raw
}

// consciousnessPenalty は疲労・空腹による意識低下量の合計を返す
func consciousnessPenalty(world w.World, entity ecs.Entity) int {
	penalty := 0
	if world.Components.Fatigue.Has(entity) {
		penalty += world.Components.Fatigue.Get(entity).ConsciousnessPenalty()
	}
	if world.Components.Hunger.Has(entity) {
		penalty += gc.HungerConsciousnessPenalty(world.Components.Hunger.Get(entity).GetLevel())
	}
	return penalty
}

// ConsciousnessSources は意識を下げる怪我以外の要因、すなわち疲労・空腹の内訳を返す。
// Effects タブの Consciousness 行の詳細がこの導出を読む
func ConsciousnessSources(world w.World, entity ecs.Entity) []gc.ProficiencySource {
	var srcs []gc.ProficiencySource
	if world.Components.Fatigue.Has(entity) {
		f := world.Components.Fatigue.Get(entity)
		if v := f.ConsciousnessPenalty(); v != 0 {
			srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceFatigue, Fatigue: f.GetLevel(), Value: -v})
		}
	}
	if world.Components.Hunger.Has(entity) {
		level := world.Components.Hunger.Get(entity).GetLevel()
		if v := gc.HungerConsciousnessPenalty(level); v != 0 {
			srcs = append(srcs, gc.ProficiencySource{Kind: gc.SourceHunger, Hunger: level, Value: -v})
		}
	}
	return srcs
}
