package balance

import (
	"github.com/kijimaD/ruins/internal/oapi"
)

// viabilityCeiling は viable とみなす死亡確率の上限。これ以下ならその武器で概ね生き延びられる。
// 設計仮説で、素手の死亡確率より十分低い値に置く。
const viabilityCeiling = 0.10

// viabilityDistinctFloor は viable な武器群が symmetric でない、すなわち選択に意味があるとみなす
// 決着ターンの幅の下限。これ未満だとどれを選んでも同じで、Pfau のいう拒否される symmetry になる。
const viabilityDistinctFloor = 0.5

// ViabilitySummary は武器ロスターの健全性を要約する。全選択肢が等価な symmetry を避け、どれも使えるが差がある
// viability を良しとする。viable が多く罠が少なく viable どうしに差があるほど健全。
type ViabilitySummary struct {
	Day          int
	Total        int     // 評価した近接武器の数
	Viable       int     // 死亡確率が viabilityCeiling 以下の武器数
	Traps        int     // 素手より弱い、すなわち劣化量が負の武器数
	ViableTTKMin float64 // viable な武器の決着ターンの最小
	ViableTTKMax float64 // viable な武器の決着ターンの最大
	Ceiling      float64 // 判定に使った死亡確率の上限
}

// ViableRate は評価した武器のうち viable の割合を返す。
func (s ViabilitySummary) ViableRate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Viable) / float64(s.Total)
}

// Distinct は viable な武器群が symmetric でないかを返す。決着ターンの幅が下限を超えれば選択に意味がある。
func (s ViabilitySummary) Distinct() bool {
	return s.ViableTTKMax-s.ViableTTKMin >= viabilityDistinctFloor
}

// WeaponViability は経過日 day の廃墟プールに対する武器ロスターの viability を要約する。要素制限の
// 劣化量計算を再利用し、viable 数・罠数・viable どうしの決着ターンの幅を集計する。
func WeaponViability(master oapi.Raws, enemyTableName string, day int) (ViabilitySummary, error) {
	values, _, err := WeaponRestrictionValues(master, enemyTableName, day)
	if err != nil {
		return ViabilitySummary{}, err
	}
	// Inf で初期化して最後にゼロへ戻す方式は、上書き漏れで Inf が外へ出る壊れ方をする。
	// viable が0件なら min/max はゼロ値のまま、1件目で両方をその値に据えてから広げる。
	s := ViabilitySummary{Day: day, Total: len(values), Ceiling: viabilityCeiling}
	first := true
	for _, v := range values {
		if v.Degradation < 0 {
			s.Traps++
		}
		if v.DeathProbWith <= viabilityCeiling {
			s.Viable++
			if first || v.ExpTurnsWith < s.ViableTTKMin {
				s.ViableTTKMin = v.ExpTurnsWith
			}
			if first || v.ExpTurnsWith > s.ViableTTKMax {
				s.ViableTTKMax = v.ExpTurnsWith
			}
			first = false
		}
	}
	return s, nil
}
