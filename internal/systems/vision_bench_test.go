package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// 視界再計算 calculateTileVisibilityWithDistance の実寸を測るベンチ。半径は実ゲームと同じ
// VisionRadiusTiles=24。荒れ地は壁が無くレイが最後まで伸びる最悪ケース、市街地は建物で
// レイが早期終了しつつ blockIndex 参照が重いケース。視界更新は毎ターンここを丸ごと引き直すので、
// 1回のコストが体感に直結する。回帰の番人として置く。

// benchPlayerTile はベンチのプレイヤータイル座標。maxTileDistance=26 を足しても正になるよう離す。
const benchPlayerTile = 60

// benchPlayerPos はプレイヤーのワールドピクセル座標(タイル中心)を返す。
func benchPlayerPos() consts.Coord[consts.WorldPixel] {
	c := benchPlayerTile*int(consts.TileSize) + int(consts.TileSize)/2
	return consts.Coord[consts.WorldPixel]{X: consts.WorldPixel(c), Y: consts.WorldPixel(c)}
}

// cityBlockIndex は市街地を模した blockIndex を作る。5×5 の壁枠の建物を 8 タイル間隔で敷き、
// レイが途中の壁でよく止まる状況を再現する。
func cityBlockIndex() map[gc.GridElement]bool {
	idx := map[gc.GridElement]bool{}
	lo, hi := benchPlayerTile-30, benchPlayerTile+30
	for by := lo; by < hi; by += 8 {
		for bx := lo; bx < hi; bx += 8 {
			for dy := range 5 {
				for dx := range 5 {
					if dx != 0 && dx != 4 && dy != 0 && dy != 4 {
						continue // 外周だけ壁にする
					}
					x, y := bx+dx, by+dy
					if x == benchPlayerTile && y == benchPlayerTile {
						continue // プレイヤー位置は空ける
					}
					idx[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
				}
			}
		}
	}
	return idx
}

func benchVision(b *testing.B, block map[gc.GridElement]bool) {
	b.Helper()
	pos := benchPlayerPos()
	radius := consts.WorldPixel(consts.VisionRadiusTiles) * consts.TileSize
	b.ResetTimer()
	for range b.N {
		calculateTileVisibilityWithDistance(pos, radius, block)
	}
}

func BenchmarkVisionRecompute_荒れ地(b *testing.B) {
	benchVision(b, map[gc.GridElement]bool{})
}

func BenchmarkVisionRecompute_市街地(b *testing.B) {
	benchVision(b, cityBlockIndex())
}
