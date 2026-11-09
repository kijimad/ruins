package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// EffectiveBodyFuncs は消費側が読む身体機能を返す。保存された不調と、疲労・空腹から組み立てた不調を
// 合わせて BodyFuncs へ畳む
func EffectiveBodyFuncs(world w.World, entity ecs.Entity) gc.BodyFuncs {
	needConds := DerivedConditions(world, entity)
	if world.Components.HealthStatus.Has(entity) {
		return world.Components.HealthStatus.Get(entity).BodyFuncs(world.Resources.ConditionDefs, needConds...)
	}
	// HealthStatus 非所持は怪我を持たない
	return (&gc.HealthStatus{}).BodyFuncs(world.Resources.ConditionDefs, needConds...)
}

// DerivedConditions は疲労・空腹の量から過労・栄養失調の不調を読み取り時に組み立てる。
// 保存された疲労・空腹の不調は無く、量を変えた時点で次の読みに自動反映される。怪我と同じ不調として同じ経路を通る
func DerivedConditions(world w.World, entity ecs.Entity) []gc.HealthCondition {
	var conds []gc.HealthCondition
	if world.Components.Fatigue.Has(entity) {
		if sev, ok := world.Components.Fatigue.Get(entity).FatigueSeverity(world.Resources.NeedDefs); ok {
			conds = append(conds, gc.HealthCondition{Type: gc.ConditionExhaustion, Severity: sev})
		}
	}
	if world.Components.Hunger.Has(entity) {
		if sev, ok := world.Resources.NeedDefs.HungerSeverity(world.Components.Hunger.Get(entity).GetLevel(world.Resources.NeedDefs)); ok {
			conds = append(conds, gc.HealthCondition{Type: gc.ConditionMalnutrition, Severity: sev})
		}
	}
	return conds
}

// ConditionName は不調の訳した表示名を返す。定義が無ければ id をそのまま返す
func ConditionName(world w.World, ct gc.ConditionType) string {
	def, ok := world.Resources.ConditionDefs.Def(ct)
	if !ok {
		return string(ct)
	}
	return T(world, def.Name)
}

// ConditionDescription は不調の訳した概要説明を返す。定義が無ければ空文字を返す
func ConditionDescription(world w.World, ct gc.ConditionType) string {
	def, ok := world.Resources.ConditionDefs.Def(ct)
	if !ok {
		return ""
	}
	return T(world, def.Description)
}
