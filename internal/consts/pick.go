package consts

// Roller は [0, 1) の小数を1つ返す。*rand.Rand はそのまま満たす。
type Roller interface {
	Float64() float64
}

// HashRoll はハッシュ値を [0, 1) の小数として引く。座標などから決定的に抽選するときに使う。
type HashRoll uint64

// Float64 は上位53ビットを [0, 1) へ写す。
func (h HashRoll) Float64() float64 {
	return float64(h>>11) / float64(1<<53)
}

// Pick は重みに比例して entries から1つ選び、その添字を返す。重みが0以下の要素は選ばない。
// 候補が空、または重みの合計が0なら false を返す。
func Pick[T any](r Roller, entries []T, weight func(T) float64) (int, bool) {
	var total float64
	last := -1
	for i, e := range entries {
		if w := weight(e); w > 0 {
			total += w
			last = i
		}
	}
	if last < 0 {
		return 0, false
	}
	x := r.Float64() * total
	var cum float64
	for i, e := range entries {
		if w := weight(e); w > 0 {
			cum += w
			if x < cum {
				return i, true
			}
		}
	}
	// Float64 が1に十分近いと、丸めで x が total と等しくなる
	return last, true
}

// PickUniform は entries から一様に1つ選び、その添字を返す。空なら false を返す。
func PickUniform[T any](r Roller, entries []T) (int, bool) {
	return Pick(r, entries, func(T) float64 { return 1 })
}
