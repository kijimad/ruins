package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// proficiencyInputs は効果倍率の導出に使う入力を集める。Skills が無ければ nil を返す。
// caps は疲労・空腹による意識低下を畳んだ実効身体機能で、命中の畳み込みが読む
func proficiencyInputs(world w.World, entity ecs.Entity) (skills *gc.Skills, abils *gc.Abilities, caps gc.BodyCapacities) {
	if world.Components.Skills.Has(entity) {
		skills = world.Components.Skills.Get(entity)
	}
	if world.Components.Abilities.Has(entity) {
		abils = world.Components.Abilities.Get(entity)
	}
	caps = EffectiveCapacities(world, entity)
	return skills, abils, caps
}

// ProficiencyValue は key の効果倍率を都度計算して返す。保存済みの値ではなく
// Skills・Abilities・実効身体機能から読み取り時に導出する。
// 適用と表示の両方がこの1関数を読むので、両者は同じ値になる。
// Skills が無ければ等倍を返し、呼び出し側の存在ガードは不要
func ProficiencyValue(world w.World, entity ecs.Entity, key gc.ProficiencyKey) consts.Percent {
	skills, abils, caps := proficiencyInputs(world, entity)
	if skills == nil {
		return consts.PercentBase
	}
	return gc.CalcProficiencyValue(skills, abils, caps, key)
}

// ProficiencySources は key の効果倍率の内訳を都度計算して返す。詳細モーダルの表示用。
// Skills が無ければ空を返す
func ProficiencySources(world w.World, entity ecs.Entity, key gc.ProficiencyKey) []gc.ProficiencySource {
	skills, abils, caps := proficiencyInputs(world, entity)
	if skills == nil {
		return nil
	}
	return gc.CalcProficiencySources(skills, abils, caps, key)
}
