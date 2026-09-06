package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// EffectiveBodyFuncs は消費側が読む身体機能を返す。怪我・病気は HealthStatus の不調から、
// 疲労・空腹は量から読み取り時に導出した意識低下として、まとめて BodyFuncs へ畳む。
// 意識は master 乗数として局所機能へ一度だけ掛かる。疲労・空腹は保存しないのでズレようがない
func EffectiveBodyFuncs(world w.World, entity ecs.Entity) gc.BodyFuncs {
	penalty := consciousnessPenalty(world, entity)
	if world.Components.HealthStatus.Has(entity) {
		return world.Components.HealthStatus.Get(entity).BodyFuncsWith(penalty)
	}
	// HealthStatus 非所持は怪我を持たない。疲労・空腹の意識低下だけを空の身体機能へ畳む
	return (&gc.HealthStatus{}).BodyFuncsWith(penalty)
}

// consciousnessPenalty は疲労・空腹の量から意識低下量を都度導出する。保存された不調でなく量を直読みするので、
// 量を変えた時点で自動的に反映される
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
