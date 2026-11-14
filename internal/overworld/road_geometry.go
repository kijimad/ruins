package overworld

import (
	"cmp"
	"slices"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// 道の幾何の唯一の出典。道が「隣接リージョンの当選集落を L 字で結び、塀に囲まれた区域は門を通る」という規則をここ一箇所に置き、
// 舗装 road.go・散布回避 scatter.go・マクロ地図 road_overlay.go の3消費者が同じ分解を各粒度で解釈する。
// 分解を1箇所にすることで、地図に出る道と実際に舗装される道が構造的に一致し続ける。

// roadOrient は道の一辺の向き。実体は文字列で、%v やテスト失敗の出力に種別名が出る。
type roadOrient string

const (
	orientHorizontal roadOrient = "horizontal" // 行を固定して列方向へ伸びる
	orientVertical   roadOrient = "vertical"   // 列を固定して行方向へ伸びる
)

// roadSeg は道の一辺。座標は絶対タイルで、fixed は道の中心線。
type roadSeg struct {
	orient roadOrient
	fixed  consts.Tile // orient=horizontal なら固定する Y、vertical なら固定する X
	lo, hi consts.Tile // 可変軸の範囲。lo<=hi で端点を含む
}

// span は lo と hi を小さい順に並べた辺を作る
func span(orient roadOrient, fixed, a, b consts.Tile) roadSeg {
	return roadSeg{orient: orient, fixed: fixed, lo: min(a, b), hi: max(a, b)}
}

// chunkCenter はチャンク c の中心のタイルを返す
func chunkCenter(c consts.Chunk, size consts.Tile) consts.Tile {
	return c.Tiles(size) + size/2
}

// roadSegments は当選集落 a から b への道を水平辺・垂直辺の2区間で返す。分解規則の唯一の出典で、
// 水平辺は始点 a の行、垂直辺は終点 b の列、角は (b.X, a.Y)。各辺はチャンクの中心を通る。端点は min/max で
// 正規化するので、a と b の東西の大小に依らず同じ2辺になる。
func roadSegments(a, b consts.Coord[consts.Chunk], chunkW, chunkH consts.Tile) []roadSeg {
	return []roadSeg{
		span(orientHorizontal, chunkCenter(a.Y, chunkH), chunkCenter(a.X, chunkW), chunkCenter(b.X, chunkW)),
		span(orientVertical, chunkCenter(b.X, chunkW), chunkCenter(a.Y, chunkH), chunkCenter(b.Y, chunkH)),
	}
}

// roadPath は当選集落 a から b への道を返す。roadSegments の垂直辺が塀に囲まれた区域を横切るなら、区域の手前の行で
// 北の門の列へ折れて門へ入り、南の門から出て区域の向こうの行で元の列へ戻る。区域の中は区域が街路を敷くので辺を置かない。
// 門は区域の外周の街路の帯に開くので、門の列では帯の幅の真ん中を通す
func roadPath(raws rawschema.Raws, runSeed uint64, a, b consts.Coord[consts.Chunk], cols consts.Chunk, chunkW, chunkH consts.Tile) []roadSeg {
	segs := roadSegments(a, b, chunkW, chunkH)
	vertical := segs[1]
	x, y := vertical.fixed, vertical.lo
	out := []roadSeg{segs[0]}
	for _, r := range fencedDistrictsBetween(raws, runSeed, min(a.Y, b.Y), max(a.Y, b.Y), cols) {
		north, south := gateColumns(raws, r)
		top := r.anchor.Y.Tiles(chunkH)
		bottom := (r.anchor.Y + r.h).Tiles(chunkH) - 1
		lane := func(col consts.Chunk) consts.Tile { return col.Tiles(chunkW) + urbanStreetW/2 }
		if y < top {
			before := chunkCenter(r.anchor.Y-1, chunkH)
			out = append(out,
				span(orientVertical, x, y, before),
				span(orientHorizontal, before, x, lane(north)),
				span(orientVertical, lane(north), before, top),
			)
		}
		if vertical.hi <= bottom {
			return out
		}
		after := chunkCenter(r.anchor.Y+r.h, chunkH)
		out = append(out,
			span(orientVertical, lane(south), bottom, after),
			span(orientHorizontal, after, lane(south), x),
		)
		y = after
	}
	return append(out, span(orientVertical, x, y, vertical.hi))
}

// fencedDistrictsBetween は絶対チャンク行 top から bottom に掛かる、塀に囲まれた区域を北から順に返す
func fencedDistrictsBetween(raws rawschema.Raws, runSeed uint64, top, bottom, cols consts.Chunk) []districtRegion {
	var out []districtRegion
	for _, d := range districtsFrom(raws) {
		if d.row.Fence == nil || !*d.row.Fence {
			continue
		}
		spacing := consts.Chunk(d.row.Spacing)
		for ry := consts.FloorDiv(top, spacing); ry <= consts.FloorDiv(bottom, spacing); ry++ {
			r := d.regionIn(runSeed, ry, cols)
			if r.anchor.Y > bottom || r.anchor.Y+r.h-1 < top {
				continue
			}
			if got, ok := districtAt(raws, runSeed, r.anchor, cols); ok && got == r {
				out = append(out, r)
			}
		}
	}
	slices.SortFunc(out, func(a, b districtRegion) int { return cmp.Compare(a.anchor.Y, b.anchor.Y) })
	return out
}

// crossingRoads はチャンク c を横切りうる道の端点対を返す。c を横切りうるのは (r-1,r) と (r,r+1) を
// 結ぶ2本だけ。road.go の舗装と scatter.go の回避が同じ結線を共有する。ホットパスで呼ぶので、
// スライスを確保せず固定長配列で返す。集落は Y 方向のリージョンに並ぶので region は c.Y から引く。
func crossingRoads(runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) [2][2]consts.Coord[consts.Chunk] {
	r := consts.FloorDiv(c.Y, settlementPlacement.Spacing)
	var pairs [2][2]consts.Coord[consts.Chunk]
	for i := range 2 {
		pr := r - 1 + consts.Chunk(i)
		pairs[i][0] = settlementPlacement.WinnerOf(runSeed, pr, cols)
		pairs[i][1] = settlementPlacement.WinnerOf(runSeed, pr+1, cols)
	}
	return pairs
}
