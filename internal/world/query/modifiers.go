package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// Modifiers はエンティティの効果倍率を都度計算して返す。保存済みの値ではなく
// Skills・Abilities・HealthStatus から読み取り時に導出する。
// Skills が無ければ全て等倍のビューを返すので、呼び出し側の存在ガードは不要
func Modifiers(world w.World, entity ecs.Entity) *gc.CharModifiers {
	if !world.Components.Skills.Has(entity) {
		return gc.CalcCharModifiers(gc.NewSkills(), nil, nil)
	}
	skills := world.Components.Skills.Get(entity)
	var abils *gc.Abilities
	if world.Components.Abilities.Has(entity) {
		abils = world.Components.Abilities.Get(entity)
	}
	var hs *gc.HealthStatus
	if world.Components.HealthStatus.Has(entity) {
		hs = world.Components.HealthStatus.Get(entity)
	}
	return gc.CalcCharModifiers(skills, abils, hs)
}
