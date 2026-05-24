package balance

import (
	"sort"
)

// Percentile はソート済みスライスからパーセンタイル値を返す。
// p は 0.0〜1.0 の範囲で指定する
func Percentile(sorted []int, p float64) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// Median はスライスの中央値を返す
func Median(values []int) int {
	if len(values) == 0 {
		return 0
	}
	s := make([]int, len(values))
	copy(s, values)
	sort.Ints(s)
	return Percentile(s, 0.5)
}

// Mean はスライスの平均値を返す
func Mean(values []int) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// RunStats はN回のランの統計をまとめる
type RunStats struct {
	Results []RunResult
}

// MedianDepth は到達深度の中央値を返す
func (s RunStats) MedianDepth() int {
	depths := make([]int, len(s.Results))
	for i, r := range s.Results {
		depths[i] = r.ReachedDepth
	}
	return Median(depths)
}

// hpPercentile は指定深度でのHPパーセンタイルを算出する汎用ヘルパー。
// selector は各 RunResult から対象のHP mapを返す
func (s RunStats) hpPercentile(depth int, selector func(RunResult) map[int]int, p float64) int {
	var hps []int
	for _, r := range s.Results {
		if hp, ok := selector(r)[depth]; ok {
			hps = append(hps, hp)
		}
	}
	if len(hps) == 0 {
		return 0
	}
	sort.Ints(hps)
	return Percentile(hps, p)
}

func postHealHP(r RunResult) map[int]int { return r.HPByDepth }
func preHealHP(r RunResult) map[int]int  { return r.HPBeforeHealByDepth }
func hungerMap(r RunResult) map[int]int  { return r.HungerByDepth }

// HPAtDepth は指定深度での残HPスライスを返す
func (s RunStats) HPAtDepth(depth int) []int {
	var hps []int
	for _, r := range s.Results {
		if hp, ok := r.HPByDepth[depth]; ok {
			hps = append(hps, hp)
		}
	}
	return hps
}

// MedianHP は指定深度での残HPの中央値を返す
func (s RunStats) MedianHP(depth int) int {
	return s.hpPercentile(depth, postHealHP, 0.5)
}

// P5HP は指定深度での下位5%の残HPを返す
func (s RunStats) P5HP(depth int) int {
	return s.hpPercentile(depth, postHealHP, 0.05)
}

// P95HP は指定深度での上位95%の残HPを返す
func (s RunStats) P95HP(depth int) int {
	return s.hpPercentile(depth, postHealHP, 0.95)
}

// MedianHPBeforeHeal は指定深度での回復前HPの中央値を返す
func (s RunStats) MedianHPBeforeHeal(depth int) int {
	return s.hpPercentile(depth, preHealHP, 0.5)
}

// P5HPBeforeHeal は指定深度での回復前HP下位5%を返す
func (s RunStats) P5HPBeforeHeal(depth int) int {
	return s.hpPercentile(depth, preHealHP, 0.05)
}

// P95HPBeforeHeal は指定深度での回復前HP上位95%を返す
func (s RunStats) P95HPBeforeHeal(depth int) int {
	return s.hpPercentile(depth, preHealHP, 0.95)
}

// SuddenDeathRate は指定深度での突然死率を返す。
// この深度に到達したランのうち、この深度で死亡した割合
func (s RunStats) SuddenDeathRate(depth int) float64 {
	if depth < 1 {
		depth = 1
	}
	survived := 0
	diedHere := 0
	for _, r := range s.Results {
		if r.ReachedDepth >= depth {
			survived++
			if r.Died && r.ReachedDepth == depth {
				diedHere++
			}
		}
	}
	if survived == 0 {
		return 0
	}
	return float64(diedHere) / float64(survived)
}

// WeaponDistribution は指定深度での武器使用分布を返す
func (s RunStats) WeaponDistribution(depth int) map[string]int {
	dist := make(map[string]int)
	for _, r := range s.Results {
		if name, ok := r.WeaponByDepth[depth]; ok {
			dist[name]++
		}
	}
	return dist
}

// MedianHunger は指定深度での空腹度の中央値を返す
func (s RunStats) MedianHunger(depth int) int {
	return s.hpPercentile(depth, hungerMap, 0.5)
}

// P5Hunger は指定深度での空腹度の下位5%を返す
func (s RunStats) P5Hunger(depth int) int {
	return s.hpPercentile(depth, hungerMap, 0.05)
}

// P95Hunger は指定深度での空腹度の上位95%を返す
func (s RunStats) P95Hunger(depth int) int {
	return s.hpPercentile(depth, hungerMap, 0.95)
}

// DeathRate は全体の死亡率を返す
func (s RunStats) DeathRate() float64 {
	if len(s.Results) == 0 {
		return 0
	}
	died := 0
	for _, r := range s.Results {
		if r.Died {
			died++
		}
	}
	return float64(died) / float64(len(s.Results))
}
