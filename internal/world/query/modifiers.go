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
	base := gc.CalcModifierValue(skills, abils, hs, key)
	if _, delta, ok := fatigueAccuracyDelta(world, entity, key, int(base)); ok {
		return consts.Percent(int(base) + delta)
	}
	return base
}

// fatigueAccuracyDelta は武器命中への疲労の畳み込みを返す。命中キーで疲労を持つときだけ ok=true。
// pre は疲労適用前の値。level は内訳表示用の段階、delta は加法差分。
// ModifierValue と ModifierSources が同じこの導出を読むので、値と内訳は一致する
func fatigueAccuracyDelta(world w.World, entity ecs.Entity, key gc.ModifierKey, pre int) (level gc.FatigueLevel, delta int, ok bool) {
	if !gc.IsWeaponAccuracyKey(key) || !world.Components.Fatigue.Has(entity) {
		return "", 0, false
	}
	fatigue := world.Components.Fatigue.Get(entity)
	post := fatigue.Penalty().AccuracyMul.ApplyInt(pre)
	return fatigue.GetLevel(), post - pre, true
}

// ModifierSources は key の効果倍率の内訳を都度計算して返す。詳細モーダルの表示用。
// Skills が無ければ空を返す
func ModifierSources(world w.World, entity ecs.Entity, key gc.ModifierKey) []gc.ModifierSource {
	skills, abils, hs, ok := modifierInputs(world, entity)
	if !ok {
		return nil
	}
	sources := gc.CalcModifierSources(skills, abils, hs, key)
	base := gc.CalcModifierValue(skills, abils, hs, key)
	if level, delta, ok := fatigueAccuracyDelta(world, entity, key, int(base)); ok && delta != 0 {
		sources = append(sources, gc.ModifierSource{Kind: gc.SourceFatigue, Fatigue: level, Value: delta})
	}
	return sources
}
