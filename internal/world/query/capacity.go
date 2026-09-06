package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// EffectiveBodyFuncs は消費側が読む身体機能を返す。怪我・病気は HealthStatus の保存 condition から、
// 疲労・空腹は量から読み取り時に materialize した need condition として、まとめて BodyFuncs へ畳む。
// 意識は master 乗数として局所機能へ一度だけ掛かる。need condition は保存しないのでズレようがない
func EffectiveBodyFuncs(world w.World, entity ecs.Entity) gc.BodyFuncs {
	needConds := DerivedConditions(world, entity)
	if world.Components.HealthStatus.Has(entity) {
		return world.Components.HealthStatus.Get(entity).BodyFuncsWith(needConds)
	}
	// HealthStatus 非所持は怪我を持たない。need condition だけを空の身体機能へ畳む
	return (&gc.HealthStatus{}).BodyFuncsWith(needConds)
}

// DerivedConditions は疲労・空腹の量から過労・栄養失調の condition を読み取り時に materialize する。
// 保存された need condition は無く、量を変えた時点で次の読みに自動反映される。怪我と同じ condition として funnel へ入る
func DerivedConditions(world w.World, entity ecs.Entity) []gc.HealthCondition {
	var conds []gc.HealthCondition
	if world.Components.Fatigue.Has(entity) {
		if sev, ok := world.Components.Fatigue.Get(entity).FatigueSeverity(); ok {
			conds = append(conds, gc.HealthCondition{Type: gc.ConditionExhaustion, Severity: sev})
		}
	}
	if world.Components.Hunger.Has(entity) {
		if sev, ok := gc.HungerSeverity(world.Components.Hunger.Get(entity).GetLevel()); ok {
			conds = append(conds, gc.HealthCondition{Type: gc.ConditionMalnutrition, Severity: sev})
		}
	}
	return conds
}
