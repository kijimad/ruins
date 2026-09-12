package overworld

import "github.com/kijimaD/ruins/internal/consts"

// 道の幾何の唯一の出典。道が「隣接リージョンの当選集落を L 字で結ぶ」という規則をここ一箇所に置き、
// 舗装 road.go・散布回避 scatter.go・マクロ地図 road_overlay.go の3消費者が同じ分解を各粒度で解釈する。
// 分解を1箇所にすることで、地図に出る道と実際に舗装される道が構造的に一致し続ける。

// roadOrient は道の一辺の向き。実体は文字列で、%v やテスト失敗の出力に種別名が出る。
type roadOrient string

const (
	orientHorizontal roadOrient = "horizontal" // 行を固定して列方向へ伸びる
	orientVertical   roadOrient = "vertical"   // 列を固定して行方向へ伸びる
)

// roadSeg は道の一辺。
type roadSeg struct {
	orient roadOrient
	fixed  consts.Chunk // orient=horizontal なら固定する行、vertical なら固定する列
	lo, hi consts.Chunk // 可変軸の範囲。lo<=hi で端点を含む
}

// roadSegments は当選集落 a から b への道を水平辺・垂直辺の2区間で返す。分解規則の唯一の出典で、
// 水平辺は始点 a の行、垂直辺は終点 b の列、角は (b.X, a.Y)。端点は min/max で正規化するので、
// a と b の東西の大小に依らず同じ2辺になる。
func roadSegments(a, b consts.Coord[consts.Chunk]) [2]roadSeg {
	return [2]roadSeg{
		{orient: orientHorizontal, fixed: a.Y, lo: min(a.X, b.X), hi: max(a.X, b.X)},
		{orient: orientVertical, fixed: b.X, lo: min(a.Y, b.Y), hi: max(a.Y, b.Y)},
	}
}

// tileSpan は辺を帯の1チャンク寸法でタイル座標へ移す。固定軸のタイルと可変軸の範囲を返し、各軸を
// チャンク中心へ寄せる。集落中心が当選チャンクのど真ん中に来るのに合わせる。
func (s roadSeg) tileSpan(chunkW, chunkH consts.Tile) (fixed, lo, hi consts.Tile) {
	if s.orient == orientHorizontal {
		// 固定は行→Y、可変は列→X
		return s.fixed.Tiles(chunkH) + chunkH/2, s.lo.Tiles(chunkW) + chunkW/2, s.hi.Tiles(chunkW) + chunkW/2
	}
	// 固定は列→X、可変は行→Y
	return s.fixed.Tiles(chunkW) + chunkW/2, s.lo.Tiles(chunkH) + chunkH/2, s.hi.Tiles(chunkH) + chunkH/2
}

// crossingRoads はチャンク c を横切りうる道の端点対を返す。c を横切りうるのは (r-1,r) と (r,r+1) を
// 結ぶ2本だけ。road.go の舗装と scatter.go の回避が同じ結線を共有する。ホットパスで呼ぶので、
// スライスを確保せず固定長配列で返す。集落は Y 方向のリージョンに並ぶので region は c.Y から引く。
func crossingRoads(runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk) [2][2]consts.Coord[consts.Chunk] {
	r := floorDiv(c.Y, settlementPlacement.Spacing)
	var pairs [2][2]consts.Coord[consts.Chunk]
	for i := range 2 {
		pr := r - 1 + consts.Chunk(i)
		pairs[i][0] = settlementPlacement.WinnerOf(runSeed, pr, cols)
		pairs[i][1] = settlementPlacement.WinnerOf(runSeed, pr+1, cols)
	}
	return pairs
}
