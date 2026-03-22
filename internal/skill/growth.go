package skill

import gc "github.com/kijimaD/ruins/internal/components"

// スキル成長の定数
const (
	baseExp       = 10 // 基本獲得経験値
	abilBonus     = 5  // 能力値1あたりの成長速度ボーナス（%）
	decayPerLevel = 20 // スキル値1あたりの減衰率（%）
)

// GainExp はスキルに経験値を加算する。スキルアップしたらtrueを返す。
// abilityValueは対応する能力値で、高いほど獲得経験値が増える。
// 式: exp = baseExp * (100 + abilValue*5) / 100 * 100 / (100 + currentValue*20)
func GainExp(s *gc.Skill, abilityValue int) bool {
	growthSpeed := 100 + abilityValue*abilBonus
	decay := 100 + s.Value*decayPerLevel

	exp := baseExp * growthSpeed / 100 * 100 / decay
	if exp < 1 {
		exp = 1
	}
	s.Exp.Current += exp

	if s.Exp.Current >= s.Exp.Max {
		s.Exp.Current -= s.Exp.Max
		s.Value++
		return true
	}

	return false
}
