package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
)

// 草の茂みの濃淡を決める定数
const (
	// grassPatchCell は茂みの大きさを決める格子の間隔。ノイズの値をこの間隔の格子点に置き、間を補間する
	grassPatchCell consts.Tile = 8
	// grassNoiseChannel は茂みのノイズを他の抽選と無相関にするハッシュチャネル
	grassNoiseChannel uint64 = 0x67726173735f6e7a // "grass_nz"
)

// grassPatchFactor は絶対タイル座標 p の草の密度倍率を返す。値ノイズ n の 3n² で、平均はおよそ0.9
func grassPatchFactor(seed uint64, p consts.Coord[consts.Tile]) float64 {
	n := valueNoise(seed, p, grassPatchCell)
	return 3 * n * n
}

// valueNoise は絶対タイル座標 p の値ノイズを 0..1 で返す。間隔 cell の格子点に seed から値を置き、間を滑らかに補間する。
// 座標の純関数なので、チャンクの境目でも途切れない
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

// placesGrass は絶対タイル座標 p に草を置くかを、密度 density と茂みの倍率から決める
func placesGrass(seed uint64, density float64, p consts.Coord[consts.Tile]) bool {
	return consts.HashRoll(consts.Mix2D(seed, p.X, p.Y)).Float64() < density*grassPatchFactor(seed^grassNoiseChannel, p)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func smoothstep(t float64) float64 {
	return t * t * (3 - 2*t)
}
