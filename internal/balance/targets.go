package balance

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// TargetCheck はドメイン横断のスカラー指標を目標帯と照合した1件。戦闘は日次カーブで内外を持つので
// ここには入れず、日で変わらないスカラー指標だけを扱う。目標帯は設計仮説でプレイで見直す。
type TargetCheck struct {
	Domain string  // ドメイン名
	Metric string  // 指標名と単位
	Value  float64 // 現状の導出値
	Lo     float64 // 目標帯の下限
	Hi     float64 // 目標帯の上限
}

// InRange は現状値が目標帯 [Lo, Hi] に収まるかを返す。
func (c TargetCheck) InRange() bool {
	return c.Value >= c.Lo && c.Value <= c.Hi
}

// DomainTargets は各ドメインのスカラー指標を現状値で評価し目標帯と照合する。目標帯は「静的下限がここに収まれば実プレイは
// 少なくともこれだけ快適」という下限側の仮説。戦闘は日次カーブで別管理する。経済の loot 収入は到達層数依存の
// scenario 指標でゲートに向かないので、ここには含めず TestBaselineSnapshot_探索収入 で追う。
func DomainTargets() []TargetCheck {
	p := DefaultParams()
	return []TargetCheck{
		{"生存・飢え", "DaysUntilHungerEmpty(日)", DaysUntilHungerEmpty(p), 0.8, 1.3},
		{"生存・疲労", "SleepTimeFraction", SleepTimeFraction(p), 0.20, 0.33},
		{"生存・寒さ", "TurnsToHypothermia(0℃)", TurnsToHypothermia(0), 8, 20},
		{"物流", "DriveRangeAllFuel(OIL満載,タイル)", DriveRangeAllFuel(oapi.OIL, consts.CubeWeightCapacityKg), 800, 1200},
		{"進行・成長", "AttacksToSkillLevel(能力0,Lv30)", float64(AttacksToSkillLevel(0, 30)), 1000, 2500},
	}
}
