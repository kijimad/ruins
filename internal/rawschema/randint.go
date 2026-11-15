package rawschema

import "math/rand/v2"

// Roll は Base に 0 以上 Spread 未満の乱数を足す。Spread は1以上であること
func (r RandInt) Roll(rng *rand.Rand) int {
	return r.Base + rng.IntN(r.Spread)
}

// Mean は Roll の期待値を返す
func (r RandInt) Mean() float64 {
	return float64(r.Base) + float64(r.Spread-1)/2
}
