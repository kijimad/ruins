package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/oapi"
)

// exchangePerturb は交換レートの求解で基準つまみに与える固定摂動。+10% を打ち消すのに相手を
// どれだけ動かすかを見る。感度行列の摂動幅とそろえる。
const exchangePerturb = 1.1

// exchangeSolveRange は相手つまみを探す倍率の範囲と反復回数。基準の +10% を打ち消すのに要する
// 相手の変化は多くの場合この範囲に収まる。範囲外なら両立不能として解けない。
const (
	exchangeSolveLo    = 0.2
	exchangeSolveHi    = 5.0
	exchangeSolveIters = 60
)

// ExchangeCell は「基準つまみを+10%したとき、相手つまみをこれだけ変えると同じメトリクスに
// 据え置ける」相互補償の1件。符号込みの変化率で、負なら相手を減らして打ち消すことを表す。
type ExchangeCell struct {
	Compensator string  // 打ち消しに動かす相手つまみ
	PctChange   float64 // 相手つまみの変化率。基準+10%あたり
	OK          bool    // 与えた範囲で解けたか。両立不能なら false
}

// KnobExchange は1つの基準つまみを+10%したときの、同じメトリクスを保つための各相手つまみの補償。
type KnobExchange struct {
	Knob  string
	Cells []ExchangeCell
}

// MetricExchange は1つのメトリクスについての交換レート表。そのメトリクスを動かすつまみだけが
// 行と列に並ぶ。2つ以上動かすつまみがなければ交換の余地がなく、表そのものを作らない。
type MetricExchange struct {
	Metric string
	Knobs  []string       // 行・列に並ぶつまみ名。基準を+10%した順
	Rows   []KnobExchange // 各基準つまみの補償
}

// ExchangeRates はメトリクスごとの交換レート表を返す。あるつまみを+10%したとき、同じメトリクスを
// 元へ戻すには別のどのつまみをどれだけ動かせばよいかを、実式を二分探索で解いて求める。感度の比
// -eA/eB は線形近似だが、戦闘のように丸めで非線形なメトリクスでは解いた方が正確なので直接解く。
func ExchangeRates(master oapi.Raws) []MetricExchange {
	knobs := knobRegistry()
	base := metricsAt(master, DefaultParams())
	out := make([]MetricExchange, 0, len(SensitivityMetricNames))
	for i, metric := range SensitivityMetricNames {
		movers := metricMovers(master, knobs, base[i], i)
		if len(movers) < 2 {
			continue
		}
		names := make([]string, len(movers))
		rows := make([]KnobExchange, len(movers))
		for r, a := range movers {
			names[r] = a.Name
			cells := make([]ExchangeCell, len(movers))
			for c, b := range movers {
				if a.Name == b.Name {
					cells[c] = ExchangeCell{Compensator: b.Name}
					continue
				}
				pct, ok := solveExchange(master, i, a, b, base[i])
				cells[c] = ExchangeCell{Compensator: b.Name, PctChange: pct, OK: ok}
			}
			rows[r] = KnobExchange{Knob: a.Name, Cells: cells}
		}
		out = append(out, MetricExchange{Metric: metric, Knobs: names, Rows: rows})
	}
	return out
}

// metricMovers は idx 番目のメトリクスを +10% 摂動で動かすつまみだけを抽出する。動かさないつまみは
// そのメトリクスの交換に参加できないので除く。
func metricMovers(master oapi.Raws, knobs []Knob, base float64, idx int) []Knob {
	movers := make([]Knob, 0, len(knobs))
	for _, k := range knobs {
		p := DefaultParams()
		*k.Ptr(&p) *= exchangePerturb
		// 戦力比など非整数メトリクスの FP 誤差で誤判定しないよう、相対許容で動いたか見る。
		if math.Abs(metricValue(master, p, idx)-base) > 1e-9*math.Max(1, math.Abs(base)) {
			movers = append(movers, k)
		}
	}
	return movers
}

// solveExchange は基準つまみ a を+10%した状態で、メトリクス idx を target へ戻す相手つまみ b の
// 倍率を二分探索で解き、その変化率を返す。範囲内で解けなければ ok=false を返す。
func solveExchange(master oapi.Raws, idx int, a, b Knob, target float64) (float64, bool) {
	eval := func(fB float64) float64 {
		p := DefaultParams()
		*a.Ptr(&p) *= exchangePerturb
		*b.Ptr(&p) *= fB
		return metricValue(master, p, idx)
	}
	fB, ok := SolveScalar(eval, target, exchangeSolveLo, exchangeSolveHi, exchangeSolveIters)
	if !ok {
		return 0, false
	}
	return (fB - 1) * 100, true
}
