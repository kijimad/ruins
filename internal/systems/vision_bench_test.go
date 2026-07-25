package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// 視界再計算の実寸を測るベンチ。単一トリガ(毎ターン全クリア=毎回フレッシュなキャッシュで
// 再計算)と、2段(静止時はキャッシュ保持でヒット)のコスト差を出す。半径は実ゲームと同じ
// VisionRadiusTiles=24。荒れ地は壁が無くレイが最後まで伸びる最悪ケース、市街地は建物で
// レイが早期終了する軽いケースを両方測る。

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

// warmCache は一度計算してヒット状態にしたキャッシュを返す。静止で2ターン目以降を模す。
func warmCache(pos consts.Coord[consts.WorldPixel], radius consts.WorldPixel, block map[gc.GridElement]bool) map[raycastCacheKey]bool {
	c := make(map[raycastCacheKey]bool)
	calculateTileVisibilityWithDistance(pos, radius, c, block)
	return c
}

func benchVision(b *testing.B, block map[gc.GridElement]bool, keepCache bool) {
	b.Helper()
	pos := benchPlayerPos()
	radius := consts.WorldPixel(consts.VisionRadiusTiles) * consts.TileSize

	if keepCache {
		// 2段: 静止中はキャッシュを保持し、毎ターンヒットで引く
		cache := warmCache(pos, radius, block)
		b.ResetTimer()
		for range b.N {
			calculateTileVisibilityWithDistance(pos, radius, cache, block)
		}
		return
	}
	// 単一トリガ: 毎ターン全クリア。毎回フレッシュなキャッシュで全レイを歩き直す
	b.ResetTimer()
	for range b.N {
		cache := make(map[raycastCacheKey]bool)
		calculateTileVisibilityWithDistance(pos, radius, cache, block)
	}
}

func BenchmarkVisionRecompute_荒れ地_単一トリガ(b *testing.B) {
	benchVision(b, map[gc.GridElement]bool{}, false)
}

func BenchmarkVisionRecompute_荒れ地_2段保持(b *testing.B) {
	benchVision(b, map[gc.GridElement]bool{}, true)
}

func BenchmarkVisionRecompute_市街地_単一トリガ(b *testing.B) {
	benchVision(b, cityBlockIndex(), false)
}

func BenchmarkVisionRecompute_市街地_2段保持(b *testing.B) {
	benchVision(b, cityBlockIndex(), true)
}
