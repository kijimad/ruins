package overworld

import "github.com/kijimaD/ruins/internal/consts"

// valueNoise は絶対タイル座標 p の値ノイズを [0, 1) で返す。間隔 cell の格子点に乱数を置き、間を滑らかに補間する。
// 座標の純関数なので、チャンクの境目をまたいでも値が途切れない
func valueNoise(seed uint64, p consts.Coord[consts.Tile], cell consts.Tile) float64 {
	cx, cy := consts.FloorDiv(p.X, cell), consts.FloorDiv(p.Y, cell)
	fx := smoothstep(float64(p.X-cx*cell) / float64(cell))
	fy := smoothstep(float64(p.Y-cy*cell) / float64(cell))
	corner := func(dx, dy consts.Tile) float64 {
		return consts.HashRoll(consts.Mix2D(seed, cx+dx, cy+dy)).Float64()
	}
	top := lerp(corner(0, 0), corner(1, 0), fx)
	bottom := lerp(corner(0, 1), corner(1, 1), fx)
	return lerp(top, bottom, fy)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func smoothstep(t float64) float64 {
	return t * t * (3 - 2*t)
}
