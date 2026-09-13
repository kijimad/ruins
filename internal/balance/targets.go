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

// DomainTargets は各ドメインのスカラー指標を現状値で評価し、目標帯との照合を返す。
// 目標帯は「静的下限がこの範囲に収まれば実プレイは少なくともこれだけ快適」という下限側の仮説。
// 回復や成長を足した実プレイは楽側へ振れるので、帯は厳しめの下限で引く。
func DomainTargets() []TargetCheck {
	return []TargetCheck{
		{"生存・飢え", "DaysUntilHungerEmpty(日)", DaysUntilHungerEmpty(), 0.8, 1.3},
		{"生存・疲労", "SleepTimeFraction", SleepTimeFraction(), 0.20, 0.33},
		{"生存・寒さ", "TurnsToHypothermia(0℃)", TurnsToHypothermia(0), 8, 20},
		{"物流", "DriveRangeAllFuel(OIL満載,タイル)", DriveRangeAllFuel(oapi.OIL, consts.CubeWeightCapacityKg), 800, 1200},
		{"経済", "AuctionTakeHomeRate(価値200,1kg)", AuctionTakeHomeRate(200, 1), 0.5, 0.85},
		{"進行・成長", "AttacksToSkillLevel(能力0,Lv30)", float64(AttacksToSkillLevel(0, 30)), 1000, 2500},
	}
}
