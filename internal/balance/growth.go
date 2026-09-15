package balance

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/skill"
)

// AttacksToSkillLevel は能力値 abilityValue でスキルを Lv0 から targetLevel へ上げるのに要する
// 攻撃回数を返す。成長は1攻撃ごとに減衰しながら経験値が入り、スキル値が上がるほど遅くなる。
// 実物の skill.GainExp を反復適用して数えるので成長式を balance 内に再実装しない。
// targetLevel が 1..MaxLevel の外なら 0 を返す。exp は常に 1 以上なので上限未満なら必ず有限回で到達する。
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

// SkillDamagePercent は素手スキルが skillLevel のときの近接ダメージ倍率を、実システムの生の Percent で
// 返す。攻撃時に baseDamage へ掛かる熟練度倍率を components.CalcProficiencyValue から引く。能力・体調は
// 中立にしてスキル値だけの寄与を見る。倍率適用は実ゲームと同じ ApplyInt の切り捨てで行うため、丸め前の
// Percent をそのまま渡せるこの形を単一出典にする。
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
