package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// modifierInputs は効果倍率の導出に使う入力コンポーネントを集める。
// 各コンポーネントは不在なら nil を返す。スキル基準の倍率は skills を要るが、
// 能力値ベースの倍率は skills が無くても abils から算出できる
func modifierInputs(world w.World, entity ecs.Entity) (skills *gc.Skills, abils *gc.Abilities, hs *gc.HealthStatus) {
	if world.Components.Skills.Has(entity) {
		skills = world.Components.Skills.Get(entity)
	}
	if world.Components.Abilities.Has(entity) {
		abils = world.Components.Abilities.Get(entity)
	}
	if world.Components.HealthStatus.Has(entity) {
		hs = world.Components.HealthStatus.Get(entity)
	}
	return skills, abils, hs
}

// ModifierValue は key の効果倍率を都度計算して返す。保存済みの値ではなく
// Skills・Abilities・HealthStatus から読み取り時に導出する。
// 適用と表示の両方がこの1関数を読むので、両者は同じ値になる。
// スキル基準の倍率で Skills が無ければ等倍を返し、呼び出し側の存在ガードは不要
func ModifierValue(world w.World, entity ecs.Entity, key gc.ModifierKey) consts.Percent {
	skills, abils, hs := modifierInputs(world, entity)
	if gc.KeyRequiresSkill(key) && skills == nil {
		return consts.PercentBase
	}
	total := int(gc.CalcModifierValue(skills, abils, hs, key))
	for _, s := range statusSources(world, entity, key, total) {
		total += s.Value
	}
	return consts.Percent(total)
}

// statusSources は状態由来の内訳を返す。spec 由来のスキル・能力値の内訳とは別に、疲労・空腹・睡眠が
// キーへ寄与する。ModifierValue と ModifierSources が同じこの導出を読むので値と内訳がずれない。
// 回復・行動速度は加算の寄与、命中は base に対する疲労の乗算畳み込み
func statusSources(world w.World, entity ecs.Entity, key gc.ModifierKey, base int) []gc.ModifierSource {
	if key == gc.ModRecovery {
		return recoverySources(world, entity)
	}
	if key == gc.ModActionSpeed {
		return actionSpeedSources(world, entity)
	}
	if level, delta, ok := fatigueAccuracyDelta(world, entity, key, base); ok && delta != 0 {
		return []gc.ModifierSource{{Kind: gc.SourceFatigue, Fatigue: level, Value: delta}}
	}
	return nil
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
// スキル基準の倍率で Skills が無ければ空を返す
func ModifierSources(world w.World, entity ecs.Entity, key gc.ModifierKey) []gc.ModifierSource {
	skills, abils, hs := modifierInputs(world, entity)
	if gc.KeyRequiresSkill(key) && skills == nil {
		return nil
	}
	sources := gc.CalcModifierSources(skills, abils, hs, key)
	base := int(gc.CalcModifierValue(skills, abils, hs, key))
	return append(sources, statusSources(world, entity, key, base)...)
}
