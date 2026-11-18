package consts

// Mix64 は splitmix64 の仕上げで x を撹拌する
func Mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}

// Mix2D は seed と2次元の座標 (x, y) を撹拌した値を返す。x と y に別の奇数を掛けるので転置した座標が衝突せず、
// 撹拌するので隣り合う座標の値も無相関になる
func Mix2D[T ~int](seed uint64, x, y T) uint64 {
	return Mix64(seed + uint64(x)*0x9E3779B97F4A7C15 + uint64(y)*0xC2B2AE3D27D4EB4F)
}
