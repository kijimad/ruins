package balance

import (
	"maps"
	"math"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// ruinsDay20PowerRatio は現行 master の廃墟 day20 の戦力比を返す。探索と感度の目的関数。
func ruinsDay20PowerRatio(master oapi.Raws) float64 {
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	if err != nil {
		return 0
	}
	weapon, err := LoadWeaponFromItem(master, BaselineWeapon)
	if err != nil {
		return 0
	}
	curve, err := DifficultyCurve(master, player, weapon, BaselineAreaTable, BaselineDays)
	if err != nil || len(curve) == 0 {
		return 0
	}
	return curve[len(curve)-1].PowerRatio
}

// withScaledMeleeDamage は itemID の近接ダメージを factor 倍した状態で fn を呼び、後で元へ戻す。master の item は
// backing array を共有するので値を退避・復元する。並行変更は競合するので、使うなら master を呼び出しごとに独立させる。
func withScaledMeleeDamage(master oapi.Raws, itemID string, factor float64, fn func()) {
	items := raw.PtrSlice(master.Items)
	for i := range items {
		if items[i].Id == itemID && items[i].Melee != nil {
			orig := items[i].Melee.Damage
			// 丸めで反映する。切り捨てだと factor=1.1 で奇数値が変化せず摂動が無反応になる
			items[i].Melee.Damage = int(math.Round(float64(orig) * factor))
			fn()
			items[i].Melee.Damage = orig
			return
		}
	}
	fn()
}

// SolveScalar は eval が単調な区間 [lo,hi] で eval(x)=target となる x を二分探索で返す。端点が挟まなければ ok=false。
// iters が0以下なら精緻化せず区間中点を返す。
func SolveScalar(eval func(float64) float64, target, lo, hi float64, iters int) (float64, bool) {
	flo, fhi := eval(lo), eval(hi)
	if (flo-target)*(fhi-target) > 0 {
		return 0, false
	}
	for range iters {
		mid := (lo + hi) / 2
		fm := eval(mid)
		if (flo-target)*(fm-target) <= 0 {
			hi = mid
		} else {
			lo, flo = mid, fm
		}
	}
	return (lo + hi) / 2, true
}

// withScaledMeleeDamages は複数アイテムの近接ダメージを同時に factor 倍して fn を呼び、後で全て戻す。単変数版の
// 多変数化で、並行変更が競合する点も同じ。
func withScaledMeleeDamages(master oapi.Raws, factors map[string]float64, fn func()) {
	items := raw.PtrSlice(master.Items)
	type saved struct {
		idx  int
		orig int
	}
	restore := make([]saved, 0, len(factors))
	for i := range items {
		if items[i].Melee == nil {
			continue
		}
		factor, ok := factors[items[i].Id]
		if !ok {
			continue
		}
		restore = append(restore, saved{idx: i, orig: items[i].Melee.Damage})
		items[i].Melee.Damage = int(math.Round(float64(items[i].Melee.Damage) * factor))
	}
	fn()
	for _, s := range restore {
		items[s.idx].Melee.Damage = s.orig
	}
}

// ParetoPoint は多目的探索の1点。決定変数の武器ダメージ倍率と、代表日ごとの目標戦力比からの逸脱を持つ。
type ParetoPoint struct {
	Factors    map[string]float64 // 決定変数。武器 id ごとの近接ダメージ倍率
	Deviations []float64          // 目的。repDays と同順の |PowerRatio(day) - TargetPowerRatio(day)|
}

// combatDeviations は現行 master の廃墟カーブで、代表日ごとの目標戦力比からの逸脱の絶対値を返す。
// すべての代表日で導出できたときだけ ok を true にする。
func combatDeviations(master oapi.Raws, repDays []int) ([]float64, bool) {
	player, err := LoadCombatantFromMember(master, BaselinePlayer)
	if err != nil {
		return nil, false
	}
	weapon, err := LoadWeaponFromItem(master, BaselineWeapon)
	if err != nil {
		return nil, false
	}
	maxDay := 0
	for _, d := range repDays {
		if d > maxDay {
			maxDay = d
		}
	}
	curve, err := DifficultyCurve(master, player, weapon, BaselineAreaTable, maxDay)
	if err != nil {
		return nil, false
	}
	devs := make([]float64, len(repDays))
	for i, d := range repDays {
		if d < 1 || d > len(curve) {
			return nil, false
		}
		devs[i] = math.Abs(curve[d-1].PowerRatio - TargetPowerRatio(d))
	}
	return devs, true
}

// gridFactors は knobs の各つまみに levels の各水準を割り当てた全組み合わせを返す。格子探索の候補集合。
func gridFactors(knobs []string, levels []float64) []map[string]float64 {
	combos := []map[string]float64{{}}
	for _, knob := range knobs {
		next := make([]map[string]float64, 0, len(combos)*len(levels))
		for _, base := range combos {
			for _, lv := range levels {
				m := make(map[string]float64, len(base)+1)
				maps.Copy(m, base)
				m[knob] = lv
				next = append(next, m)
			}
		}
		combos = next
	}
	return combos
}

// dominates は a のすべての目的が b 以下で、少なくとも1つで真に小さいとき真を返す。逸脱は小さいほど良い。
func dominates(a, b []float64) bool {
	strictly := false
	for i := range a {
		if a[i] > b[i] {
			return false
		}
		if a[i] < b[i] {
			strictly = true
		}
	}
	return strictly
}

// ParetoFront は knobs×levels の格子で武器ダメージ倍率を動かし、代表日 repDays の目標逸脱の非劣な点集合を返す。
// 複数日のトレードオフは逸脱ベクトルの非劣集合で初めて意味を持つ。withScaledMeleeDamages が master を書き換えるので
// この格子ループは並行化しない。並行なら格子点ごとに独立した master をロードすること。
func ParetoFront(master oapi.Raws, knobs []string, levels []float64, repDays []int) []ParetoPoint {
	combos := gridFactors(knobs, levels)
	points := make([]ParetoPoint, 0, len(combos))
	for _, factors := range combos {
		var devs []float64
		var ok bool
		withScaledMeleeDamages(master, factors, func() {
			devs, ok = combatDeviations(master, repDays)
		})
		if !ok {
			continue
		}
		points = append(points, ParetoPoint{Factors: factors, Deviations: devs})
	}
	front := make([]ParetoPoint, 0, len(points))
	for i := range points {
		dominated := false
		for j := range points {
			if i != j && dominates(points[j].Deviations, points[i].Deviations) {
				dominated = true
				break
			}
		}
		if !dominated {
			front = append(front, points[i])
		}
	}
	return front
}
