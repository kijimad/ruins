package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// modifierInputs は効果倍率の導出に使う入力コンポーネントを集める。
// Skills が無ければ ok=false で、呼び出し側は等倍・内訳なしとして扱う
func modifierInputs(world w.World, entity ecs.Entity) (skills *gc.Skills, abils *gc.Abilities, hs *gc.HealthStatus, ok bool) {
	if !world.Components.Skills.Has(entity) {
		return nil, nil, nil, false
	}
	skills = world.Components.Skills.Get(entity)
	if world.Components.Abilities.Has(entity) {
		abils = world.Components.Abilities.Get(entity)
	}
	if world.Components.HealthStatus.Has(entity) {
		hs = world.Components.HealthStatus.Get(entity)
	}
	return skills, abils, hs, true
}

// ModifierValue は key の効果倍率を都度計算して返す。保存済みの値ではなく
// Skills・Abilities・HealthStatus から読み取り時に導出する。
// 適用と表示の両方がこの1関数を読むので、両者は同じ値になる。
// Skills が無ければ等倍を返し、呼び出し側の存在ガードは不要
func ModifierValue(world w.World, entity ecs.Entity, key gc.ModifierKey) consts.Percent {
	skills, abils, hs, ok := modifierInputs(world, entity)
	if !ok {
		return consts.PercentBase
	}
	return gc.CalcModifierValue(skills, abils, hs, key)
}

// ModifierSources は key の効果倍率の内訳を都度計算して返す。詳細モーダルの表示用。
// Skills が無ければ空を返す
func ModifierSources(world w.World, entity ecs.Entity, key gc.ModifierKey) []gc.ModifierSource {
	skills, abils, hs, ok := modifierInputs(world, entity)
	if !ok {
		return nil
	}
	return gc.CalcModifierSources(skills, abils, hs, key)
}
