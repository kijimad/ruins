package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// EffectiveBodyFuncs は消費側が読む身体機能を返す。怪我・病気は HealthStatus に保存された不調から、
// 疲労・空腹は量から読み取り時に組み立てた不調として、まとめて BodyFuncs へ畳む。
// 意識は全体乗数として局所機能へ一度だけ掛かる。疲労・空腹の不調は保存しないのでズレようがない
func EffectiveBodyFuncs(world w.World, entity ecs.Entity) gc.BodyFuncs {
	needConds := DerivedConditions(world, entity)
	if world.Components.HealthStatus.Has(entity) {
		return world.Components.HealthStatus.Get(entity).BodyFuncsWith(needConds)
	}
	// HealthStatus 非所持は怪我を持たない。疲労・空腹の不調だけを空の身体機能へ畳む
	return (&gc.HealthStatus{}).BodyFuncsWith(needConds)
}

// DerivedConditions は疲労・空腹の量から過労・栄養失調の不調を読み取り時に組み立てる。
// 保存された疲労・空腹の不調は無く、量を変えた時点で次の読みに自動反映される。怪我と同じ不調として同じ経路を通る
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
