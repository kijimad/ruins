package overworld

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// errRoomSizeMismatch は部屋の絵の大きさがチャンクから帯を引いた大きさと一致しないときに返す
var errRoomSizeMismatch = errors.New("room template size does not match the chunk")

// buildingLot は区域の中の建物の区画。区域の中の座標 anchor から w×h チャンク広がり、施設を1つ建てる
type buildingLot struct {
	facility string
	anchor   consts.Coord[consts.Chunk]
	w, h     consts.Chunk
	interior rawschema.FacilityInterior
	seed     uint64
}

// lotsOf は区域 r を建物の区画で埋める。左上のチャンクから順に、未割り当てのチャンクの地区で施設を引き、
// 施設の大きさの候補を重みで引く。区画は同じ地区で未割り当てのチャンクだけに収まるまで小さくする
func lotsOf(raws rawschema.Raws, cat zoneCatalog, r districtRegion) []buildingLot {
	facilities := map[string]rawschema.Facility{}
	for _, f := range raw.PtrSlice(raws.Facilities) {
		facilities[f.Id] = f
	}
	taken := map[consts.Coord[consts.Chunk]]bool{}
	free := func(x, y consts.Chunk, z rawschema.Zone) bool {
		c := consts.Coord[consts.Chunk]{X: x, Y: y}
		return x < r.w && y < r.h && !taken[c] && r.zoneAt(c) == z
	}
	fits := func(x, y, w, h consts.Chunk, z rawschema.Zone) bool {
		for dy := range h {
			for dx := range w {
				if !free(x+dx, y+dy, z) {
					return false
				}
			}
		}
		return true
	}

	var lots []buildingLot
	for y := range r.h {
		for x := range r.w {
			l := consts.Coord[consts.Chunk]{X: x, Y: y}
			if taken[l] {
				continue
			}
			z := r.zoneAt(l)
			// 施設抽選は建物幾何と別の乱数ストリームにして、片方を変えても他方が動かないようにする。
			// ストリーム識別子 0x1 は施設抽選、0x2 は建物幾何と敵配置。renderUrbanChunk と揃える
			frng := rand.New(rand.NewPCG(ChunkSeed2D(r.seed, x, y), 0x1))
			f := facilities[rollFacilityInZone(frng, cat, z, r.span())]
			w, h := consts.Chunk(1), consts.Chunk(1)
			if fps := raw.PtrSlice(f.Footprints); len(fps) > 0 {
				i, _ := consts.Pick(frng, fps, func(fp rawschema.FacilityFootprint) float64 { return float64(fp.Weight) })
				w, h = consts.Chunk(fps[i].W), consts.Chunk(fps[i].H)
			}
			for !fits(x, y, w, h, z) {
				if w >= h && w > 1 {
					w--
				} else {
					h--
				}
			}
			interior := rawschema.FacilityInteriorRooms
			if f.Interior != nil {
				interior = *f.Interior
			}
			for dy := range h {
				for dx := range w {
					taken[consts.Coord[consts.Chunk]{X: x + dx, Y: y + dy}] = true
				}
			}
			lots = append(lots, buildingLot{facility: f.Id, anchor: l, w: w, h: h, interior: interior, seed: ChunkSeed2D(r.seed, x, y)})
		}
	}
	return lots
}

// lotAt は区域の中の座標 l を含む区画を返す。lotsOf が区域を隙間なく埋めるので必ず見つかる
func lotAt(lots []buildingLot, l consts.Coord[consts.Chunk]) buildingLot {
	for _, b := range lots {
		if l.X >= b.anchor.X && l.X < b.anchor.X+b.w && l.Y >= b.anchor.Y && l.Y < b.anchor.Y+b.h {
			return b
		}
	}
	panic(fmt.Sprintf("no lot covers %v", l))
}

// roomEdges は区画の中の座標 l のチャンクの、北・東・南・西の辺の種類を返す。区画の外へ向く辺は壁で、
// 北の辺の真ん中の列にだけ入口の扉を開ける。区画の内側の辺は、hall なら吹き抜け、rooms なら迷路の扉か壁
func (b buildingLot) roomEdges(l consts.Coord[consts.Chunk]) [4]maptemplate.EdgeKind {
	var doors map[[2]consts.Coord[consts.Chunk]]bool
	if b.interior == rawschema.FacilityInteriorRooms {
		doors = b.mazeDoors()
	}
	edge := func(d consts.Coord[consts.Chunk]) maptemplate.EdgeKind {
		n := l.Add(d)
		if n.X < 0 || n.Y < 0 || n.X >= b.w || n.Y >= b.h {
			return maptemplate.EdgeWall
		}
		if b.interior == rawschema.FacilityInteriorHall {
			return maptemplate.EdgeOpen
		}
		if doors[edgeKey(l, n)] {
			return maptemplate.EdgeDoor
		}
		return maptemplate.EdgeWall
	}
	edges := [4]maptemplate.EdgeKind{
		edge(consts.Coord[consts.Chunk]{Y: -1}),
		edge(consts.Coord[consts.Chunk]{X: 1}),
		edge(consts.Coord[consts.Chunk]{Y: 1}),
		edge(consts.Coord[consts.Chunk]{X: -1}),
	}
	if l.Y == 0 && l.X == b.w/2 {
		edges[0] = maptemplate.EdgeDoor
	}
	return edges
}

// mazeDoors は区画の中のチャンクを節にした全域木の辺を返す。入口のチャンクから、区画の seed で順を混ぜた
// 深さ優先でたどる。どの部屋にも入口から扉を通って入れる
func (b buildingLot) mazeDoors() map[[2]consts.Coord[consts.Chunk]]bool {
	rng := rand.New(rand.NewPCG(b.seed, 0x6))
	doors := map[[2]consts.Coord[consts.Chunk]]bool{}
	seen := map[consts.Coord[consts.Chunk]]bool{}
	var visit func(c consts.Coord[consts.Chunk])
	visit = func(c consts.Coord[consts.Chunk]) {
		seen[c] = true
		dirs := []consts.Coord[consts.Chunk]{{Y: -1}, {X: 1}, {Y: 1}, {X: -1}}
		rng.Shuffle(len(dirs), func(i, j int) { dirs[i], dirs[j] = dirs[j], dirs[i] })
		for _, d := range dirs {
			n := c.Add(d)
			if n.X < 0 || n.Y < 0 || n.X >= b.w || n.Y >= b.h || seen[n] {
				continue
			}
			doors[edgeKey(c, n)] = true
			visit(n)
		}
	}
	visit(consts.Coord[consts.Chunk]{X: b.w / 2})
	return doors
}

// edgeKey は隣り合う2チャンクの辺を、向きに依らない鍵にする
func edgeKey(a, b consts.Coord[consts.Chunk]) [2]consts.Coord[consts.Chunk] {
	if a.Y < b.Y || a.Y == b.Y && a.X < b.X {
		return [2]consts.Coord[consts.Chunk]{a, b}
	}
	return [2]consts.Coord[consts.Chunk]{b, a}
}

// roomChunkCells は区画 b の中の座標 l のチャンクの部屋の絵を引き、北と西の帯と合わせてチャンク全体のマスを組む。
// 帯は区画の外周なら街路、内側なら隣の部屋との繋ぎにする。屋内のマスを判定する関数も返す
func roomChunkCells(loader *maptemplate.TemplateLoader, b buildingLot, l consts.Coord[consts.Chunk], chunkW, chunkH consts.Tile) ([][]maptemplate.MapCell, func(x, y int) bool, error) {
	edges := b.roomEdges(l)
	_, room, err := loader.FacilityRoom(b.facility, edges, ChunkSeed2D(b.seed, l.X, l.Y))
	if err != nil {
		return nil, nil, err
	}
	s := int(urbanStreetW)
	w, h := int(chunkW), int(chunkH)
	if len(room) != h-s || len(room[0]) != w-s {
		return nil, nil, fmt.Errorf("facility %q room %dx%d, chunk %dx%d: %w", b.facility, len(room[0]), len(room), w, h, errRoomSizeMismatch)
	}

	street := maptemplate.MapCell{Terrain: consts.TileNameFloor}
	cells := make([][]maptemplate.MapCell, h)
	for y := range cells {
		cells[y] = make([]maptemplate.MapCell, w)
		for x := range cells[y] {
			cells[y][x] = street
		}
	}
	for y, row := range room {
		copy(cells[y+s][s:], row)
	}
	drawRoomStrips(cells, room, b, l, edges)
	return cells, roomIndoor(l, s), nil
}

// drawRoomStrips は区画の内側に面するチャンクの北と西の帯を、隣の部屋との繋ぎとして描く。区画の外周の帯は街路のまま残す
func drawRoomStrips(cells, room [][]maptemplate.MapCell, b buildingLot, l consts.Coord[consts.Chunk], edges [4]maptemplate.EdgeKind) {
	s := int(urbanStreetW)
	h, w := len(cells), len(cells[0])
	wall := maptemplate.MapCell{Terrain: "wall"}
	floor := func(x, y int) maptemplate.MapCell { return maptemplate.MapCell{Terrain: room[y][x].Terrain} }
	mid := (w - s) / 2

	// 帯の端が区画の外に面していれば壁にして、吹き抜けの帯から建物の外へ抜けないようにする
	northEnd := func(x int) bool { return l.X == 0 && x == s || l.X == b.w-1 && x == w-1 }
	westEnd := func(y int) bool { return l.Y == 0 && y == s || l.Y == b.h-1 && y == h-1 }
	if l.Y > 0 {
		for y := range s {
			for x := s; x < w; x++ {
				cells[y][x] = wall
				if stripOpen(edges[0], x-s, mid, northEnd(x)) {
					cells[y][x] = floor(x-s, 0)
				}
			}
		}
	}
	if l.X > 0 {
		for y := s; y < h; y++ {
			for x := range s {
				cells[y][x] = wall
				if stripOpen(edges[3], y-s, mid, westEnd(y)) {
					cells[y][x] = floor(0, y-s)
				}
			}
		}
	}
	if l.X > 0 && l.Y > 0 {
		corner := wall
		if b.interior == rawschema.FacilityInteriorHall {
			corner = floor(0, 0)
		}
		for y := range s {
			for x := range s {
				cells[y][x] = corner
			}
		}
	}
}

// roomIndoor は区画の中の座標 l のチャンクで、部屋と、区画の内側に面する帯を屋内とする判定を返す
func roomIndoor(l consts.Coord[consts.Chunk], s int) func(x, y int) bool {
	return func(x, y int) bool {
		inRoom := x >= s && y >= s
		northStrip := y < s && l.Y > 0 && (x >= s || l.X > 0)
		westStrip := x < s && l.X > 0 && (y >= s || l.Y > 0)
		return inRoom || northStrip || westStrip
	}
}

// stripOpen は辺の種類 kind の帯で、辺に沿った位置 i のマスが床かを返す。扉は真ん中の2マス、吹き抜けは
// 区画の外に面する端 end を除く全て
func stripOpen(kind maptemplate.EdgeKind, i, mid int, end bool) bool {
	switch kind {
	case maptemplate.EdgeOpen:
		return !end
	case maptemplate.EdgeDoor:
		return i == mid-1 || i == mid
	case maptemplate.EdgeWall:
		return false
	}
	return false
}
