package consts

// Mix64 は splitmix64 の仕上げで x を撹拌する。1違いの入力が無相関の出力になるので、座標から決定的な
// 乱数を作る土台に使う
func Mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}
