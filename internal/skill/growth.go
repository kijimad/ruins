package skill

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// growthConfig はスキル成長のバランスパラメータ
var growthConfig = struct {
	BaseExp       int // 基本獲得経験値
	AbilBonus     int // 能力値1あたりの成長速度ボーナス（%）
	DecayPerLevel int // スキル値1あたりの減衰率（%）
	MaxLevel      int // スキルの最大レベル。0の場合は上限なし
}{
	BaseExp:       10,
	AbilBonus:     5,
	DecayPerLevel: 20,
	MaxLevel:      100,
}

// GainExp はスキルに経験値を加算する。スキルアップしたらtrueを返す。
// abilityValueは対応する能力値で、高いほど獲得経験値が増える。
func GainExp(s *gc.Skill, abilityValue int) bool {
	return gainExp(s, abilityValue, growthConfig.BaseExp)
}

// GainExpScaled はBaseExpにefficiencyPct（0-100%の倍率）を適用してから経験値を加算する。
// 読書など、状況に応じて獲得量を調整する場合に使う。
func GainExpScaled(s *gc.Skill, abilityValue int, efficiencyPct consts.Percent) bool {
	baseExp := max(efficiencyPct.ApplyInt(growthConfig.BaseExp), 1)
	return gainExp(s, abilityValue, baseExp)
}

// gainExp は共通の経験値加算処理
func gainExp(s *gc.Skill, abilityValue int, baseExp int) bool {
	if growthConfig.MaxLevel > 0 && s.Value >= growthConfig.MaxLevel {
		return false
	}

	// 経験値への乗数倍率。decay は割り算で効く逆適用の倍率なのでApplyInt は使わず int のまま扱う。
	growthSpeed := consts.PercentBase + consts.Percent(abilityValue*growthConfig.AbilBonus)
	decay := 100 + s.Value*growthConfig.DecayPerLevel

	exp := max(growthSpeed.ApplyInt(baseExp)*100/decay, 1)
	s.Exp.Current += exp

	if s.Exp.Current >= s.Exp.Max {
		s.Exp.Current -= s.Exp.Max
		s.Value++
		return true
	}

	return false
}
