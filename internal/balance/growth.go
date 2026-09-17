package balance

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/skill"
)

// AttacksToSkillLevel は abilityValue でスキルを Lv0 から targetLevel へ上げるのに要する攻撃回数を返す。
// 実物の skill.GainExp を反復して数える。範囲外は 0。exp は常に1以上なので上限未満なら有限回で到達する。
func AttacksToSkillLevel(abilityValue, targetLevel int) int {
	if targetLevel <= 0 || targetLevel > skill.MaxLevel() {
		return 0
	}
	s := &gc.Skill{Exp: gc.IntPool{Max: gc.LevelUpExp}}
	attacks := 0
	for s.Value < targetLevel {
		skill.GainExp(s, abilityValue)
		attacks++
	}
	return attacks
}

// SkillLevelAfterAttacks は能力値 abilityValue で attacks 回攻撃したあとの到達スキル値を返す。
// AttacksToSkillLevel の逆で、決まった手数でどこまで伸びるかを見る。
func SkillLevelAfterAttacks(abilityValue, attacks int) int {
	s := &gc.Skill{Exp: gc.IntPool{Max: gc.LevelUpExp}}
	for range attacks {
		skill.GainExp(s, abilityValue)
	}
	return s.Value
}

// SkillDamagePercent は素手スキル skillLevel の近接ダメージ倍率を実システムの生 Percent で返す。能力・体調は
// 中立にしスキル寄与だけを見る。適用は ApplyInt の切り捨てなので、丸め前の Percent を渡せるこの形を単一出典にする。
func SkillDamagePercent(skillLevel int) consts.Percent {
	skills := gc.NewSkills()
	skills.Get(gc.SkillFist).Value = skillLevel
	return gc.CalcProficiencyValue(skills, &gc.Abilities{}, gc.HealthyBodyFuncs(), gc.WeaponDamageKey(gc.SkillFist))
}

// SkillDamageMultiplier は SkillDamagePercent を等倍1.0基準の float で見た値。表示や概算に使う。
// ダメージへの実適用は SkillDamagePercent と ApplyInt の切り捨てで行い、float の丸めに頼らない。
func SkillDamageMultiplier(skillLevel int) float64 {
	return float64(SkillDamagePercent(skillLevel)) / float64(consts.PercentBase)
}
