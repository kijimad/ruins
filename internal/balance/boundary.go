package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/oapi"
)

// 境界マージンの求解範囲と反復回数。つまみをどこまで動かすと目標帯を割るかを探す。
const (
	boundarySolveLo    = 0.05
	boundarySolveHi    = 20.0
	boundarySolveIters = 60
)

// metricBand は感度メトリクスの目標帯。has が false のメトリクスは自然な帯がなく境界監視の対象外。
// 帯は設計仮説で、DomainTargets と重なる指標はその帯にそろえる。プレイで見直す。
type metricBand struct {
	lo, hi float64
	has    bool
}

// metricBands は SensitivityMetricNames と同順の目標帯を返す。金額そのものの loot手取りは自然な帯が
// ないので監視しない。
func metricBands() []metricBand {
	b := make([]metricBand, len(SensitivityMetricNames))
	b[0] = metricBand{lo: 0.5, hi: 1.4, has: true}   // 戦力比d20。敵を進行で強化した床のハードモード帯
	b[1] = metricBand{lo: 0.5, hi: 1.0, has: true}   // 飢餓まで日数
	b[2] = metricBand{lo: 0.20, hi: 0.33, has: true} // 睡眠時間割合。DomainTargets と同じ
	b[3] = metricBand{lo: 800, hi: 1200, has: true}  // OIL航続。DomainTargets と同じ
	b[4] = metricBand{}                              // loot手取り。金額で自然な帯がない
	b[5] = metricBand{lo: 1000, hi: 2500, has: true} // Lv30攻撃数。DomainTargets と同じ
	return b
}

// BoundaryMargin は1つのつまみを動かしたとき、あるメトリクスが目標帯を割るまでの余白。凍結ゲートが
// 現在値の点を固定するのに対し、こちらは崖までの距離を測る。絶対値が小さいほど崖が近い。
type BoundaryMargin struct {
	Metric     string
	Knob       string
	NearestPct float64 // 最寄りの帯端に達するまでのつまみ変化率。符号込み
	Edge       string  // 最寄りの端。"下限" か "上限"
	OK         bool    // 与えた範囲で端に到達できたか
}

// BoundaryMargins はメトリクス×つまみごとの境界マージンを返す。各つまみを動かして目標帯の端に達する
// までの変化率を二分探索で解く。凍結ゲートの点固定を、余白の監視へ拡張する。目標帯を持つメトリクスと、
// それを動かすつまみの組だけを対象にする。
func BoundaryMargins(master oapi.Raws) []BoundaryMargin {
	knobs := knobRegistry()
	base := metricsAt(master, DefaultParams())
	bands := metricBands()
	out := make([]BoundaryMargin, 0, len(SensitivityMetricNames)*len(knobs))
	for i, name := range SensitivityMetricNames {
		if !bands[i].has {
			continue
		}
		for _, k := range metricMovers(master, knobs, base[i], i) {
			m := solveBoundary(master, i, k, bands[i])
			m.Metric = name
			m.Knob = k.Name
			out = append(out, m)
		}
	}
	return out
}

// solveBoundary はつまみ k を動かしてメトリクス idx を目標帯 band の端へ届かせる変化率を解き、両端の
// うち近い方を返す。どちらの端にも範囲内で届かなければ OK=false。
func solveBoundary(master oapi.Raws, idx int, k Knob, band metricBand) BoundaryMargin {
	eval := func(f float64) float64 {
		p := DefaultParams()
		*k.Ptr(&p) *= f
		return metricValue(master, p, idx)
	}
	edges := []struct {
		target float64
		label  string
	}{
		{band.lo, "下限"},
		{band.hi, "上限"},
	}
	best := BoundaryMargin{}
	for _, e := range edges {
		f, ok := SolveScalar(eval, e.target, boundarySolveLo, boundarySolveHi, boundarySolveIters)
		if !ok {
			continue
		}
		pct := (f - 1) * 100
		if !best.OK || math.Abs(pct) < math.Abs(best.NearestPct) {
			best = BoundaryMargin{NearestPct: pct, Edge: e.label, OK: true}
		}
	}
	return best
}
