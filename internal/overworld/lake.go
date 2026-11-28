package overworld

import (
	"errors"
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

// scatterAt は区域の中の座標 l の層が撒く散布の種類を返す。区域の中の座標だけで呼ぶ
func (r districtRegion) scatterAt(l consts.Coord[consts.Chunk]) *rawschema.EntityID {
	return r.kind.Layers[r.layerAt(l)].Scatter
}

// lakeShape は区域の湖の輪郭と桟橋。湖を描く層のチャンクを合わせた長方形に収まる楕円を、絶対タイル座標で持つ
type lakeShape struct {
	water          *rawschema.DistrictWater
	cx, cy, rx, ry float64
	seed           uint64
	piers          map[consts.Coord[consts.Tile]]bool
}

// errLakeTooSmall は湖を描く層が、輪郭の揺れと浜の幅を引くと楕円が残らないほど小さいときに返す
var errLakeTooSmall = errors.New("lake layer is too small for its shore wobble and beach width")

// lakeShapeOf は区域 r の湖の輪郭と、区画 lots から伸ばす桟橋を返す。楕円は揺れの量と浜の幅だけ内側に取り、湖が隣の層へはみ出さず、湖の層の縁に岸の陸を残す。湖が無ければ ok=false
func (r districtRegion) lakeShapeOf(lots []buildingLot, chunkW, chunkH consts.Tile) (lakeShape, bool, error) {
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
		return lakeShape{}, false, nil
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
	if s.rx <= 0 || s.ry <= 0 {
		return lakeShape{}, false, fmt.Errorf("district %q: %w", r.kind.Id, errLakeTooSmall)
	}
	for _, b := range lots {
		s.addPier(r, b.anchor.Add(b.entrance()), chunkW, chunkH)
	}
	return s, true, nil
}

// waterSide は区域の中の座標 l に接する湖の向きを、北・南・西・東の順に最初に見つけたもので返す
func (r districtRegion) waterSide(l consts.Coord[consts.Chunk]) (consts.Coord[consts.Chunk], bool) {
	for _, d := range []consts.Coord[consts.Chunk]{{Y: -1}, {Y: 1}, {X: -1}, {X: 1}} {
		if r.waterAt(l.Add(d)) != nil {
			return d, true
		}
	}
	return consts.Coord[consts.Chunk]{}, false
}

// quarterTurnsToward は北を向いた絵を向き d へ向けるのに要る左回りの90度の回数を返す
func quarterTurnsToward(d consts.Coord[consts.Chunk]) int {
	switch d {
	case consts.Coord[consts.Chunk]{X: -1}:
		return 1
	case consts.Coord[consts.Chunk]{Y: 1}:
		return 2
	case consts.Coord[consts.Chunk]{X: 1}:
		return 3
	}
	return 0
}

// addPier は区画の入口のチャンク l に置いた施設の絵の真ん中から、接する湖の側へ幅2の桟橋を伸ばす。浅瀬を越えて pierLength だけ深い水へ入った所で止める
func (s *lakeShape) addPier(r districtRegion, l consts.Coord[consts.Chunk], chunkW, chunkH consts.Tile) {
	d, ok := r.waterSide(l)
	if !ok {
		return
	}
	c := r.anchor.Add(l)
	center := consts.Coord[consts.Tile]{X: c.X.Tiles(chunkW), Y: c.Y.Tiles(chunkH)}.Add(facilityArtCenter(chunkW, chunkH))
	s.layPier(center, consts.Coord[consts.Tile]{X: consts.Tile(d.X), Y: consts.Tile(d.Y)}, max(r.w.Tiles(chunkW), r.h.Tiles(chunkH)))
}

// layPier は from から step の向きへ、輪郭の内側に幅2の板を張る。深い水へ pierLength だけ入るか、湖の中心を越えるか、
// 板を張った後に陸へ出たら止める。中心を越えたかは揺れの無い楕円で見る
func (s *lakeShape) layPier(from, step consts.Coord[consts.Tile], limit consts.Tile) {
	// 2列目を負の向きに張り、どの向きでも中心の列にそろえる
	side := consts.Coord[consts.Tile]{X: -tileAbs(step.Y), Y: -tileAbs(step.X)}
	end := float64(s.water.BeachWidth + s.water.ShallowWidth + s.water.PierLength)
	prev := math.Inf(-1)
	laid := false
	for p, n := from, consts.Tile(0); n < limit; p, n = p.Add(step), n+1 {
		depth := s.depth(p)
		inside := s.ellipseInside(p)
		if depth >= end || depth >= 0 && inside < prev || laid && depth < 0 {
			return
		}
		prev = inside
		if depth >= 0 {
			laid = true
			s.piers[p] = true
			if s.depth(p.Add(side)) >= 0 {
				s.piers[p.Add(side)] = true
			}
		}
	}
}

// depth は絶対タイル座標 p が湖の輪郭からどれだけ内側にあるかをマス数で返す。負なら輪郭の外の陸
func (s lakeShape) depth(p consts.Coord[consts.Tile]) float64 {
	return s.ellipseInside(p) + float64(s.water.ShoreWobble)*(2*valueNoise(s.seed, p, consts.Tile(s.water.ShoreCell))-1)
}

// ellipseInside は p が揺れの無い楕円の輪郭からどれだけ内側にあるかをマス数で返す
func (s lakeShape) ellipseInside(p consts.Coord[consts.Tile]) float64 {
	dx := (float64(p.X) + 0.5 - s.cx) / s.rx
	dy := (float64(p.Y) + 0.5 - s.cy) / s.ry
	return (1 - math.Hypot(dx, dy)) * min(s.rx, s.ry)
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
	raws := world.Resources.RawMaster
	shape, ok, err := r.lakeShapeOf(lotsOf(raws, zoneCatalogFrom(raws), r, spawnRowOf(world)), g.chunkW, g.chunkH)
	if err != nil || !ok {
		return err
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

func tileAbs(v consts.Tile) consts.Tile {
	return max(v, -v)
}
