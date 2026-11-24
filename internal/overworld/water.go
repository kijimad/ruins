package overworld

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// 水面の形を決める定数。湖は粗い格子のノイズ、沼は細かい格子のノイズで作り、重ねた値の高い所を水にする
const (
	// waterLakeCell は湖の大きさを決める格子の間隔
	waterLakeCell consts.Tile = 24
	// waterMarshCell は沼の水たまりの大きさを決める格子の間隔
	waterMarshCell consts.Tile = 6
	// waterMarshWeight は重ねるノイズのうち沼の割合。残りが湖
	waterMarshWeight = 0.3
	// waterShallowLevel を超えると浅瀬、waterDeepLevel を超えると深い水にする
	waterShallowLevel = 0.72
	waterDeepLevel    = 0.8
	// waterMarshChannel は沼のノイズを湖のノイズと無相関にするハッシュチャネル
	waterMarshChannel uint64 = 0x77617465725f6d73 // "water_ms"
)

// 水面のタイル
const (
	waterShallowTile = "water_shallow"
	waterDeepTile    = "water_deep"
)

// waterTileAt は絶対タイル座標 p に置く水のタイルを返す。水でなければ空文字
func waterTileAt(seed uint64, p consts.Coord[consts.Tile]) string {
	n := (1-waterMarshWeight)*valueNoise(seed, p, waterLakeCell) + waterMarshWeight*valueNoise(seed^waterMarshChannel, p, waterMarshCell)
	switch {
	case n > waterDeepLevel:
		return waterDeepTile
	case n > waterShallowLevel:
		return waterShallowTile
	}
	return ""
}

// waterFeature は荒れ地と道のチャンクの土の地面に、湖と沼の水面を置く feature。
// 形は絶対タイル座標の純関数なので、隣のチャンクへ途切れずに続く。道の舗装は土でないので残り、水を渡る道になる
type waterFeature struct{}

func (waterFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	raws := world.Resources.RawMaster
	switch chunkTypeAt(raws, runSeed, c, cols, spawnRowOf(world)) {
	case chunkWasteland, chunkRoad:
	case chunkCrashSite, chunkUrban, chunkSettlement, chunkLandmark:
		return nil
	}
	earth := earthTilesOf(raws)
	tiles := g.tiles.get()
	seed := runSeed ^ waterSalt
	for y := range g.chunkH {
		for x := range g.chunkW {
			name := waterTileAt(seed, consts.Coord[consts.Tile]{X: c.X.Tiles(g.chunkW) + x, Y: c.Y.Tiles(g.chunkH) + y})
			bl := consts.Coord[consts.Tile]{X: g.offsetX + x, Y: g.offsetY + y}
			if name == "" || !earth.at(world, tiles, bl) {
				continue
			}
			if _, err := replaceTileIndexed(world, tiles, bl, name, nil); err != nil {
				return fmt.Errorf("failed to place water (%s): %w", name, err)
			}
		}
	}
	return nil
}
