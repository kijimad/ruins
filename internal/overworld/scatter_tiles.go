package overworld

import (
	"cmp"
	"slices"

	"github.com/kijimaD/ruins/internal/consts"
)

// scatterTiles はチャンク w×h の外周1タイルを除いた内側から、accept を満たすタイルを count 枚選ぶ。
// 並びは seed とタイル座標で決まるので、同じ入力なら同じタイルを返す。外周を除くので、チャンクの継ぎ目には置かない
func scatterTiles(w, h consts.Tile, accept func(consts.Coord[consts.Tile]) bool, seed uint64, count int) []consts.Coord[consts.Tile] {
	if count <= 0 {
		return nil
	}
	type scored struct {
		pos   consts.Coord[consts.Tile]
		score uint64
	}
	var cands []scored
	for y := consts.Tile(1); y < h-1; y++ {
		for x := consts.Tile(1); x < w-1; x++ {
			t := consts.Coord[consts.Tile]{X: x, Y: y}
			if accept(t) {
				cands = append(cands, scored{pos: t, score: tileJitter(seed, t) >> 11})
			}
		}
	}
	slices.SortStableFunc(cands, func(a, b scored) int {
		if a.score != b.score {
			return cmp.Compare(b.score, a.score)
		}
		if a.pos.Y != b.pos.Y {
			return cmp.Compare(a.pos.Y, b.pos.Y)
		}
		return cmp.Compare(a.pos.X, b.pos.X)
	})
	out := make([]consts.Coord[consts.Tile], min(count, len(cands)))
	for i := range out {
		out[i] = cands[i].pos
	}
	return out
}

// tileJitter は seed とタイル座標を splitmix64 の finalizer で撹拌する
func tileJitter(seed uint64, t consts.Coord[consts.Tile]) uint64 {
	x := seed
	x ^= uint64(uint32(t.X)) * 0x9E3779B97F4A7C15
	x ^= uint64(uint32(t.Y)) * 0xC2B2AE3D27D4EB4F
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return x
}
