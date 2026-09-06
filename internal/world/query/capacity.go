package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// EffectiveBodyFunctions は怪我・病気に加え、疲労・空腹による意識低下を畳んだ実効身体機能を返す。
// 速度・命中・回復がこの1関数を読むので、身体状態が行動へ効く経路が身体機能ただ1つに集約される。
// 怪我・病気は HealthStatus から、疲労・空腹は意識オフセットとして畳む
func EffectiveBodyFunctions(world w.World, entity ecs.Entity) gc.BodyFunctions {
	caps := gc.HealthyBodyFunctions()
	if world.Components.HealthStatus.Has(entity) {
		caps = world.Components.HealthStatus.Get(entity).BodyFunctions()
	}
	return caps.WithConsciousnessPenalty(consciousnessPenalty(world, entity))
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
