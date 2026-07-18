package consts

// Percent は 100 を基準(等倍)とする倍率。120 なら 1.2 倍、50 なら 0.5 倍、0 なら 0 倍。
// CharModifiers の各倍率や進行速度など「基準 100 の%」に使い、base*pct/100 の適用を
// 一元化して意味を型に載せる。
type Percent int

// PercentBase は等倍の基準値。
const PercentBase Percent = 100

// applyPercent は倍率適用の中核。式を1箇所に集約する。
// Go の / は型で意味が変わるため、int で呼べば整数除算で切り捨て、float で呼べば
// 非切り捨てになり、外皮の ApplyInt/ApplyFloat が期待する丸めが型ごとに自動で決まる。
func applyPercent[T ~int | ~float64](base T, p Percent) T {
	return base * T(p) / T(PercentBase)
}

// ApplyInt は base に倍率を適用する。整数計算のため端数は切り捨てる。
func (p Percent) ApplyInt(base int) int {
	return applyPercent(base, p)
}

// ApplyFloat は base に倍率を適用する(float64)。切り捨ては行わない。
func (p Percent) ApplyFloat(base float64) float64 {
	return applyPercent(base, p)
}
