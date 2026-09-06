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

// isStatusDerivedKey は状態コンポーネントだけで組める key かを返す。回復と行動速度は
// スキル由来の寄与を持たず、疲労・空腹・睡眠などの状態ソースだけで倍率が決まる。
// この種のキーは Skills が無くても状態ソースを算出する
func isStatusDerivedKey(key gc.ModifierKey) bool {
	return key == gc.ModRecovery || key == gc.ModActionSpeed
}

// ModifierValue は key の効果倍率を都度計算して返す。保存済みの値ではなく
// Skills・Abilities・HealthStatus から読み取り時に導出する。
// 適用と表示の両方がこの1関数を読むので、両者は同じ値になる。
// Skills が無ければ等倍を返し、呼び出し側の存在ガードは不要
func ModifierValue(world w.World, entity ecs.Entity, key gc.ModifierKey) consts.Percent {
	skills, abils, hs, ok := modifierInputs(world, entity)
	// 状態由来キーは Skills が無くても状態ソースから算出する。他キーは Skills を要る
	if !ok && !isStatusDerivedKey(key) {
		return consts.PercentBase
	}
	base := int(consts.PercentBase)
	if ok {
		base = int(gc.CalcModifierValue(skills, abils, hs, key))
	}
	total := base
	for _, s := range statusSources(world, entity, key, base) {
		total += s.Value
	}
	return consts.Percent(total)
}

// statusSources は状態由来の内訳を返す。スキル由来の内訳とは別に、疲労・空腹・睡眠・回復の VIT が
// キーへ寄与する。ModifierValue と ModifierSources が同じこの導出を読むので値と内訳がずれない。
// 回復は加算の寄与、命中は base に対する疲労の乗算畳み込み
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
// Skills が無ければ空を返す
func ModifierSources(world w.World, entity ecs.Entity, key gc.ModifierKey) []gc.ModifierSource {
	skills, abils, hs, ok := modifierInputs(world, entity)
	if !ok && !isStatusDerivedKey(key) {
		return nil
	}
	var sources []gc.ModifierSource
	base := int(consts.PercentBase)
	if ok {
		sources = gc.CalcModifierSources(skills, abils, hs, key)
		base = int(gc.CalcModifierValue(skills, abils, hs, key))
	}
	return append(sources, statusSources(world, entity, key, base)...)
}
