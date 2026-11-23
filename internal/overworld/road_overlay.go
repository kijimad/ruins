package overworld

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// マクロ地図に道を重ねるための算出。道タイルを生成せず、各チャンクをどの方角へ抜けるかをビット集合で表す。

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

// buildRoadOverlay は表示範囲の道のチャンクが道でつながる方角を、絶対チャンク座標をキーに算出する。舗装 road.go と同じく
// roadDirsAt を引くので、地図に出る道と舗装される道が一致する。生成を伴わない純関数。
func buildRoadOverlay(raws rawschema.Raws, runSeed uint64, area MacroRange, cols, spawnRow consts.Chunk) map[consts.Coord[consts.Chunk]]RoadDir {
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	for y := area.OriginY; y < area.OriginY+area.Rows; y++ {
		for x := area.OriginX; x < area.OriginX+area.Cols; x++ {
			c := consts.Coord[consts.Chunk]{X: x, Y: y}
			if dirs, _, ok := roadDirsAt(raws, runSeed, c, cols, spawnRow); ok && dirs.Any() {
				overlay[c] = dirs
			}
		}
	}
	return overlay
}
