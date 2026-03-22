package skill

import gc "github.com/kijimaD/ruins/internal/components"

// GrowthConfig はスキル成長のバランスパラメータ
type GrowthConfig struct {
	BaseExp       int // 基本獲得経験値
	AbilBonus     int // 能力値1あたりの成長速度ボーナス（%）
	DecayPerLevel int // スキル値1あたりの減衰率（%）
	MaxLevel      int // スキルの最大レベル。0の場合は上限なし
}

// DefaultGrowthConfig はデフォルトのスキル成長パラメータを返す
func DefaultGrowthConfig() GrowthConfig {
	return GrowthConfig{
		BaseExp:       10,
		AbilBonus:     5,
		DecayPerLevel: 20,
		MaxLevel:      100,
	}
}

// GainExp はスキルに経験値を加算する。スキルアップしたらtrueを返す。
// abilityValueは対応する能力値で、高いほど獲得経験値が増える。
// 式: exp = BaseExp * (100 + abilValue*AbilBonus) / 100 * 100 / (100 + currentValue*DecayPerLevel)
func GainExp(s *gc.Skill, abilityValue int, cfg GrowthConfig) bool {
	if cfg.MaxLevel > 0 && s.Value >= cfg.MaxLevel {
		return false
	}

	growthSpeed := 100 + abilityValue*cfg.AbilBonus
	decay := 100 + s.Value*cfg.DecayPerLevel

	exp := cfg.BaseExp * growthSpeed / 100 * 100 / decay
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
