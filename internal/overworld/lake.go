package overworld

import (
	"fmt"
	"math"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
)

// lakeNoiseChannel は湖の輪郭のノイズを他の抽選と無相関にするハッシュチャネル
const lakeNoiseChannel uint64 = 0x6c616b655f6e7a // "lake_nz"

// waterAt は区域の中の座標 l の層が湖を描く層なら、その水面を返す。区域の外と、湖を描かない層は nil
func (r districtRegion) waterAt(l consts.Coord[consts.Chunk]) *rawschema.DistrictWater {
	if !r.contains(r.anchor.Add(l)) {
		return nil
	}
	return r.kind.Layers[r.layerAt(l)].Water
}

// lakeShape は区域の湖の輪郭と桟橋。湖を描く層のチャンクを合わせた長方形に収まる楕円を、絶対タイル座標で持つ
type lakeShape struct {
	water          *rawschema.DistrictWater
	cx, cy, rx, ry float64
	seed           uint64
	piers          map[consts.Coord[consts.Tile]]bool
}

// lakeShapeOf は区域 r の湖の輪郭と桟橋を返す。楕円は湖が隣の層へはみ出さないよう内側に取る。湖が無ければ ok=false
func (r districtRegion) lakeShapeOf(chunkW, chunkH consts.Tile) (lakeShape, bool) {
	var water *rawschema.DistrictWater
	lo := consts.Coord[consts.Chunk]{X: r.w, Y: r.h}
	var hi consts.Coord[consts.Chunk]
	for y := range r.h {
		for x := range r.w {
			if wt := r.waterAt(consts.Coord[consts.Chunk]{X: x, Y: y}); wt != nil {
				water = wt
				lo = consts.Coord[consts.Chunk]{X: min(lo.X, x), Y: min(lo.Y, y)}
				hi = consts.Coord[consts.Chunk]{X: max(hi.X, x+1), Y: max(hi.Y, y+1)}
			}
		}
	}
	if water == nil {
		return lakeShape{}, false
	}
	x0, y0 := float64((r.anchor.X + lo.X).Tiles(chunkW)), float64((r.anchor.Y + lo.Y).Tiles(chunkH))
	x1, y1 := float64((r.anchor.X + hi.X).Tiles(chunkW)), float64((r.anchor.Y + hi.Y).Tiles(chunkH))
	margin := float64(water.ShoreWobble + water.BeachWidth)
	s := lakeShape{
		water: water,
		cx:    (x0 + x1) / 2,
		cy:    (y0 + y1) / 2,
		rx:    (x1-x0)/2 - margin,
		ry:    (y1-y0)/2 - margin,
		seed:  r.seed ^ lakeNoiseChannel,
		piers: map[consts.Coord[consts.Tile]]bool{},
	}
	for y := range r.h {
		for x := range r.w {
			if l := (consts.Coord[consts.Chunk]{X: x, Y: y}); r.builds(l) {
				s.addPier(r, l, chunkW, chunkH)
			}
		}
	}
	return s, true
}

// addPier は施設を建てたチャンク l の真ん中から、接する湖の側へ幅2の桟橋を伸ばす。浅瀬を越えて pierLength だけ深い水へ入った所で止める
func (s *lakeShape) addPier(r districtRegion, l consts.Coord[consts.Chunk], chunkW, chunkH consts.Tile) {
	for _, d := range []consts.Coord[consts.Chunk]{{Y: -1}, {Y: 1}, {X: -1}, {X: 1}} {
		if r.waterAt(l.Add(d)) == nil {
			continue
		}
		c := r.anchor.Add(l)
		center := consts.Coord[consts.Tile]{X: c.X.Tiles(chunkW) + chunkW/2, Y: c.Y.Tiles(chunkH) + chunkH/2}
		step := consts.Coord[consts.Tile]{X: consts.Tile(d.X), Y: consts.Tile(d.Y)}
		side := consts.Coord[consts.Tile]{X: step.Y, Y: step.X}
		end := float64(s.water.BeachWidth + s.water.ShallowWidth + s.water.PierLength)
		for p := center; ; p = p.Add(step) {
			depth := s.depth(p)
			if depth >= end {
				return
			}
			if depth >= 0 {
				s.piers[p] = true
				s.piers[p.Add(side)] = true
			}
		}
	}
}

// depth は絶対タイル座標 p が湖の輪郭からどれだけ内側にあるかをマス数で返す。負なら輪郭の外の陸
func (s lakeShape) depth(p consts.Coord[consts.Tile]) float64 {
	dx := (float64(p.X) + 0.5 - s.cx) / s.rx
	dy := (float64(p.Y) + 0.5 - s.cy) / s.ry
	inside := (1 - math.Hypot(dx, dy)) * min(s.rx, s.ry)
	return inside + float64(s.water.ShoreWobble)*(2*valueNoise(s.seed, p, consts.Tile(s.water.ShoreCell))-1)
}

// tileAt は絶対タイル座標 p に置く水面か桟橋のタイルを返す。輪郭の外の陸なら ok=false
func (s lakeShape) tileAt(p consts.Coord[consts.Tile]) (string, bool) {
	d := s.depth(p)
	switch {
	case s.piers[p]:
		return s.water.Pier, true
	case d < 0:
		return "", false
	case d < float64(s.water.BeachWidth):
		return s.water.Beach, true
	case d < float64(s.water.BeachWidth+s.water.ShallowWidth):
		return s.water.Shallow, true
	}
	return s.water.Deep, true
}

// renderLakeChunk はチャンク c の湖の輪郭の内側に水面と桟橋を置く。輪郭の外は下地の土を残す
func renderLakeChunk(world w.World, g chunkGeom, r districtRegion, c consts.Coord[consts.Chunk]) error {
	shape, ok := r.lakeShapeOf(g.chunkW, g.chunkH)
	if !ok {
		return nil
	}
	rows := map[string]rawschema.Tile{}
	for _, id := range []string{shape.water.Deep, shape.water.Shallow, shape.water.Beach, shape.water.Pier} {
		t, err := raw.GetTile(world.Resources.RawMaster, id)
		if err != nil {
			return fmt.Errorf("lake tile %q: %w", id, err)
		}
		rows[id] = t
	}
	origin := consts.Coord[consts.Tile]{X: c.X.Tiles(g.chunkW), Y: c.Y.Tiles(g.chunkH)}
	tiles := g.tiles.get()
	for y := range g.chunkH {
		for x := range g.chunkW {
			tile, ok := shape.tileAt(origin.Add(consts.Coord[consts.Tile]{X: x, Y: y}))
			if !ok {
				continue
			}
			if err := spawnFacilityTile(world, tiles, consts.Coord[consts.Tile]{X: g.offsetX + x, Y: g.offsetY + y}, rows[tile]); err != nil {
				return err
			}
		}
	}
	return nil
}
