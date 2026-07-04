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
	idx := max(int(float64(len(sorted)-1)*p), 0)
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

func postHealHP(r RunResult) map[int]int      { return r.HPByDepth }
func preHealHP(r RunResult) map[int]int       { return r.HPBeforeHealByDepth }
func hungerMap(r RunResult) map[int]int       { return r.HungerByDepth }
func weaponDamageMap(r RunResult) map[int]int { return r.WeaponDamageByDepth }
func killTurnsMap(r RunResult) map[int]int    { return r.AvgKillTurnsByDepth }

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

// MedianWeaponDamage は指定深度での武器ダメージの中央値を返す
func (s RunStats) MedianWeaponDamage(depth int) int {
	return s.hpPercentile(depth, weaponDamageMap, 0.5)
}

// P5WeaponDamage は指定深度での武器ダメージの下位5%を返す
func (s RunStats) P5WeaponDamage(depth int) int {
	return s.hpPercentile(depth, weaponDamageMap, 0.05)
}

// P95WeaponDamage は指定深度での武器ダメージの上位95%を返す
func (s RunStats) P95WeaponDamage(depth int) int {
	return s.hpPercentile(depth, weaponDamageMap, 0.95)
}

// MedianKillTurns は指定深度での1戦あたり平均キルターンの中央値を返す
func (s RunStats) MedianKillTurns(depth int) int {
	return s.hpPercentile(depth, killTurnsMap, 0.5)
}

// P5KillTurns は指定深度でのキルターンの下位5%を返す
func (s RunStats) P5KillTurns(depth int) int {
	return s.hpPercentile(depth, killTurnsMap, 0.05)
}

// P95KillTurns は指定深度でのキルターンの上位95%を返す
func (s RunStats) P95KillTurns(depth int) int {
	return s.hpPercentile(depth, killTurnsMap, 0.95)
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

// MedianDamagePerFloor は指定深度でのフロアあたり被ダメージの中央値を返す。
// playerMaxHP は深度1の開始HPとして使う
func (s RunStats) MedianDamagePerFloor(depth, playerMaxHP int) int {
	return s.floorDamagePercentile(depth, playerMaxHP, 0.5)
}

// MedianHealingPerFloor は指定深度でのフロアあたり回復量の中央値を返す
func (s RunStats) MedianHealingPerFloor(depth int) int {
	return s.floorHealingPercentile(depth, 0.5)
}

// floorDamagePercentile は指定深度でのフロアあたり被ダメージのパーセンタイルを返す
func (s RunStats) floorDamagePercentile(depth, playerMaxHP int, p float64) int {
	var damages []int
	for _, r := range s.Results {
		hpBefore, ok := r.HPBeforeHealByDepth[depth]
		if !ok {
			continue
		}
		entering := playerMaxHP
		if depth > 1 {
			if hp, ok := r.HPByDepth[depth-1]; ok {
				entering = hp
			} else {
				continue
			}
		}
		dmg := max(entering-hpBefore, 0)
		damages = append(damages, dmg)
	}
	if len(damages) == 0 {
		return 0
	}
	sort.Ints(damages)
	return Percentile(damages, p)
}

// floorHealingPercentile は指定深度でのフロアあたり回復量のパーセンタイルを返す
func (s RunStats) floorHealingPercentile(depth int, p float64) int {
	var heals []int
	for _, r := range s.Results {
		hpBefore, ok1 := r.HPBeforeHealByDepth[depth]
		hpAfter, ok2 := r.HPByDepth[depth]
		if !ok1 || !ok2 {
			continue
		}
		heal := max(hpAfter-hpBefore, 0)
		heals = append(heals, heal)
	}
	if len(heals) == 0 {
		return 0
	}
	sort.Ints(heals)
	return Percentile(heals, p)
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
