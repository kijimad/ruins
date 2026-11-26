package overworld

import (
	"fmt"
	"math"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
)

const (
	// lakeShoreCell は岸の線の揺れの格子の間隔。ノイズの値をこの間隔の格子点に置き、間を補間する
	lakeShoreCell consts.Tile = 6
	// lakeNoiseChannel は岸の揺れのノイズを他の抽選と無相関にするハッシュチャネル
	lakeNoiseChannel uint64 = 0x6c616b655f6e7a // "lake_nz"
)

// waterAt は区域の中の座標 l の層が水で満たす層なら、その水面を返す。区域の外と、水で満たさない層は nil
func (r districtRegion) waterAt(l consts.Coord[consts.Chunk]) *rawschema.DistrictWater {
	if !r.contains(r.anchor.Add(l)) {
		return nil
	}
	return r.kind.Layers[r.layerAt(l)].Water
}

// lakeShore は水で満たすチャンクの、陸に面する辺。rings の中心は長方形なので、斜めだけが陸の角は生じない
type lakeShore struct {
	north, south, west, east bool
}

// shoreOf は水で満たすチャンク l の陸に面する辺を、上下左右の隣のチャンクが水かどうかから導く
func (r districtRegion) shoreOf(l consts.Coord[consts.Chunk]) lakeShore {
	land := func(dx, dy consts.Chunk) bool {
		return r.waterAt(l.Add(consts.Coord[consts.Chunk]{X: dx, Y: dy})) == nil
	}
	return lakeShore{north: land(0, -1), south: land(0, 1), west: land(-1, 0), east: land(1, 0)}
}

// landDistance はチャンクの中のマス (x, y) から、陸に面する辺までのマス数の一番小さい値を返す。陸に面さなければ無限大
func (s lakeShore) landDistance(x, y, w, h consts.Tile) float64 {
	d := math.Inf(1)
	edge := func(on bool, v consts.Tile) {
		if on {
			d = min(d, float64(v))
		}
	}
	edge(s.north, y)
	edge(s.south, h-1-y)
	edge(s.west, x)
	edge(s.east, w-1-x)
	return d
}

// renderLakeChunk は区域 r の水で満たすチャンク c を描く。陸に面する辺から、浜、浅瀬、深い水の順に並べる。
// 浅瀬の幅は絶対タイル座標の値ノイズで揺らし、区域の中のどのチャンクも同じ種で揺らすので、岸の線は境目で途切れない
func renderLakeChunk(world w.World, g chunkGeom, r districtRegion, c consts.Coord[consts.Chunk]) error {
	l := r.local(c)
	water := r.waterAt(l)
	shore := r.shoreOf(l)
	origin := consts.Coord[consts.Tile]{X: c.X.Tiles(g.chunkW), Y: c.Y.Tiles(g.chunkH)}
	tiles := g.tiles.get()
	rows := map[string]rawschema.Tile{}
	for _, id := range []string{water.Deep, water.Shallow, water.Beach} {
		t, err := raw.GetTile(world.Resources.RawMaster, id)
		if err != nil {
			return fmt.Errorf("lake tile %q: %w", id, err)
		}
		rows[id] = t
	}
	for y := range g.chunkH {
		for x := range g.chunkW {
			tile := water.Deep
			if d := shore.landDistance(x, y, g.chunkW, g.chunkH); !math.IsInf(d, 1) {
				abs := origin.Add(consts.Coord[consts.Tile]{X: x, Y: y})
				shallow := float64(water.ShallowWidth) + float64(water.Wobble)*(2*valueNoise(r.seed^lakeNoiseChannel, abs, lakeShoreCell)-1)
				switch {
				case d < float64(water.BeachWidth):
					tile = water.Beach
				case d < float64(water.BeachWidth)+shallow:
					tile = water.Shallow
				}
			}
			if err := spawnFacilityTile(world, tiles, consts.Coord[consts.Tile]{X: g.offsetX + x, Y: g.offsetY + y}, rows[tile]); err != nil {
				return err
			}
		}
	}
	return nil
}
