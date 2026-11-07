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
	cx, cy := consts.FloorDiv(p.X, grassPatchCell), consts.FloorDiv(p.Y, grassPatchCell)
	fx := smoothstep(float64(p.X-cx*grassPatchCell) / float64(grassPatchCell))
	fy := smoothstep(float64(p.Y-cy*grassPatchCell) / float64(grassPatchCell))
	corner := func(dx, dy consts.Tile) float64 {
		return unitFloat(hashTileCoord(seed, consts.Coord[consts.Tile]{X: cx + dx, Y: cy + dy}))
	}
	top := lerp(corner(0, 0), corner(1, 0), fx)
	bottom := lerp(corner(0, 1), corner(1, 1), fx)
	n := lerp(top, bottom, fy)
	return 3 * n * n
}

// placesGrass は絶対タイル座標 p に草を置くかを、密度 density と茂みの倍率から決める
func placesGrass(seed uint64, density float64, p consts.Coord[consts.Tile]) bool {
	return unitFloat(hashTileCoord(seed, p)) < density*grassPatchFactor(seed^grassNoiseChannel, p)
}

// unitFloat はハッシュ値を [0, 1) の小数へ写す
func unitFloat(v uint64) float64 {
	return float64(v>>11) / float64(1<<53)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func smoothstep(t float64) float64 {
	return t * t * (3 - 2*t)
}
