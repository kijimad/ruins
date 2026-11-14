package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// マクロ地図に道を重ねるための算出。道タイルを生成せず、road_geometry.go の roadPath を
// 唯一の出典として道の辺をチャンク空間で解釈する。舗装 road.go・散布 scatter.go と同じ分解を
// 共有するので、地図に出る道と実際の道が構造的に一致する。各チャンクをどの方角へ抜けるかをビット集合で表す。

// RoadDir はチャンクを通る道が接続する方角のビット集合。線分描画で、セル中央からどの辺の
// 中点へ線を引くかを表す。0 は道なし。
type RoadDir uint8

const (
	// RoadN は上、Y-1 の隣へ抜ける。
	RoadN RoadDir = 1 << iota
	// RoadS は下、Y+1 の隣へ抜ける。
	RoadS
	// RoadE は右、X+1 の隣へ抜ける。
	RoadE
	// RoadW は左、X-1 の隣へ抜ける。
	RoadW
)

// Any は道が1方角でも接続するかを返す。0 は道なしの空集合。
func (d RoadDir) Any() bool {
	return d != 0
}

// buildRoadOverlay は表示範囲に重なる範囲の道の接続方角を、絶対チャンク座標をキーに算出する。
// road.go と同じく隣接リージョンの当選集落どうしを roadPath で結ぶ。生成を伴わない純関数で、
// 表示範囲外へはみ出すチャンクも含みうるが、読み手が表示範囲内だけを引くので無害。
func buildRoadOverlay(raws rawschema.Raws, runSeed uint64, area MacroRange, cols consts.Chunk, chunkW, chunkH consts.Tile) map[consts.Coord[consts.Chunk]]RoadDir {
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	cols = max(cols, 1)
	// 道 (pr, pr+1) が占めるチャンク列は当選集落 a.X..b.X で、a.X は pr*Spacing 以上、b.X は
	// (pr+2)*Spacing 未満に収まる。よって表示範囲 [OriginX, OriginX+Cols) に列が掛かりうる道は、表示範囲の左端の
	// 属するリージョンの1つ西から表示範囲の右端の属するリージョンまで。端の道を取りこぼさないよう、左右へ
	// scanMargin ぶんの余裕を足す。表示範囲外へ出たチャンクを印しても読み手が引かないので無害
	const scanMargin consts.Chunk = 1
	topRegion := consts.FloorDiv(area.OriginY, settlementPlacement.Spacing)
	botRegion := consts.FloorDiv(area.OriginY+area.Rows-1, settlementPlacement.Spacing)
	rLo := topRegion - 1 - scanMargin // 1つ北のリージョンの道が南へ食い込みうる
	rHi := botRegion + scanMargin
	for pr := rLo; pr <= rHi; pr++ {
		a := settlementPlacement.WinnerOf(runSeed, pr, cols)
		b := settlementPlacement.WinnerOf(runSeed, pr+1, cols)
		markRoadPath(overlay, roadPath(raws, runSeed, a, b, cols, chunkW, chunkH), chunkW, chunkH)
	}
	return overlay
}

// markRoadPath は道の辺 segs が通るチャンクへ接続方角を積む。辺の端のチャンクには端の側のビットを積まないので、
// 辺どうしが接する折れは両方のビットが積まれて L 字になり、複数の道が重なる交差はビットが増えて T・十字になる。
func markRoadPath(overlay map[consts.Coord[consts.Chunk]]RoadDir, segs []roadSeg, chunkW, chunkH consts.Tile) {
	for _, seg := range segs {
		alongSize, acrossSize := chunkW, chunkH
		if seg.orient == orientVertical {
			alongSize, acrossSize = chunkH, chunkW
		}
		fixed := consts.Chunk(consts.FloorDiv(seg.fixed, acrossSize))
		lo := consts.Chunk(consts.FloorDiv(seg.lo, alongSize))
		hi := consts.Chunk(consts.FloorDiv(seg.hi, alongSize))
		for v := lo; v <= hi; v++ {
			var cell consts.Coord[consts.Chunk]
			var bits RoadDir
			if seg.orient == orientHorizontal {
				cell = consts.Coord[consts.Chunk]{X: v, Y: fixed}
				if v > lo {
					bits |= RoadW
				}
				if v < hi {
					bits |= RoadE
				}
			} else {
				cell = consts.Coord[consts.Chunk]{X: fixed, Y: v}
				if v > lo {
					bits |= RoadN
				}
				if v < hi {
					bits |= RoadS
				}
			}
			// 退化辺、すなわち1チャンクに収まる辺はビットが 0。空の道キーを作らないよう積まない
			if bits.Any() {
				overlay[cell] |= bits
			}
		}
	}
}
