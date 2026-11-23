package overworld

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"maps"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testChunkSide consts.Tile = 24

// composeBuilding は区画 b の全てのチャンクのマスを生成と同じ planUrbanChunk で組んで、区画全体の1枚のマスにする
func composeBuilding(t *testing.T, raws rawschema.Raws, loader *maptemplate.TemplateLoader, b buildingLot) [][]maptemplate.MapCell {
	t.Helper()
	side := int(testChunkSide)
	cells := make([][]maptemplate.MapCell, int(b.h)*side)
	for y := range cells {
		cells[y] = make([]maptemplate.MapCell, int(b.w)*side)
	}
	for ly := range b.h {
		for lx := range b.w {
			c, _, err := planUrbanChunk(raws, loader, b, consts.Coord[consts.Chunk]{X: lx, Y: ly}, testChunkSide, testChunkSide, b.seed, nil, streetTiles{street: consts.TileNameRoad, verge: "grass_green"})
			require.NoError(t, err)
			for y, row := range c {
				copy(cells[int(ly)*side+y][int(lx)*side:], row)
			}
		}
	}
	return cells
}

func TestBuilding_部屋で組む建物は入口から全ての部屋へ入れる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	rules := newCellRules(raws)

	for _, fac := range loader.Facilities() {
		if !loader.HasRooms(fac) {
			continue
		}
		for _, b := range roomLots(raws, fac) {
			cells := composeBuilding(t, raws, loader, b)
			name := fmt.Sprintf("%s %dx%d seed=%d", fac, b.w, b.h, b.seed)
			assert.Empty(t, unreachableCells(cells, rules), name)
			for y, row := range cells {
				for x, c := range row {
					if rules.storage[c.Prop] {
						assert.NotEmpty(t, c.LootTable, "%s %s at %d,%d", name, c.Prop, x, y)
					}
				}
			}
		}
	}
}

// roomLots は部屋の絵で組む施設 fac が取りうる区画を返す。大きさの候補を縮めた全ての大きさを seed 30 個ずつ、施設の行の中の繋がりで組む
func roomLots(raws rawschema.Raws, fac string) []buildingLot {
	f, _ := raw.GetFacility(raws, fac)
	interior := rawschema.FacilityInteriorRooms
	if f.Interior != nil {
		interior = *f.Interior
	}
	maxW, maxH := consts.Chunk(1), consts.Chunk(1)
	for _, fp := range raw.PtrSlice(f.Footprints) {
		maxW, maxH = max(maxW, consts.Chunk(fp.W)), max(maxH, consts.Chunk(fp.H))
	}
	var out []buildingLot
	for w := consts.Chunk(1); w <= maxW; w++ {
		for h := consts.Chunk(1); h <= maxH; h++ {
			for seed := range uint64(30) {
				out = append(out, buildingLot{facility: fac, w: w, h: h, interior: interior, seed: seed})
			}
		}
	}
	return out
}

func TestFacilityRooms_区画が要る辺の組の絵を持ち辺の種類どおりに描く(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	isTall := tileIsTall(raws)
	want := map[maptemplate.EdgeKind]string{
		maptemplate.EdgeWall: "####################",
		maptemplate.EdgeDoor: "#########++#########",
		maptemplate.EdgeOpen: "..................",
	}
	for _, fac := range loader.Facilities() {
		if !loader.HasRooms(fac) {
			continue
		}
		needed := map[[4]maptemplate.EdgeKind]bool{}
		for _, b := range roomLots(raws, fac) {
			if b.seed != 0 {
				continue
			}
			for ly := range b.h {
				for lx := range b.w {
					for _, edges := range possibleRoomEdges(b, consts.Coord[consts.Chunk]{X: lx, Y: ly}) {
						needed[edges] = true
					}
				}
			}
		}
		for edges := range needed {
			_, cells, err := loader.FacilityRoom(fac, edges, 1)
			require.NoError(t, err, "%s %v", fac, edges)
			last := len(cells) - 1
			for i, k := range edges {
				got := ""
				for j := range 20 {
					c := [4]maptemplate.MapCell{cells[0][j], cells[j][last], cells[last][j], cells[j][0]}[i]
					switch {
					case c.Prop == "door":
						got += "+"
					case isTall(c.Terrain):
						got += "#"
					default:
						got += "."
					}
				}
				// 吹き抜けの辺の両端は、隣の辺が壁なら壁になるので、端を除いて比べる
				if k == maptemplate.EdgeOpen {
					got = got[1:19]
				}
				assert.Equal(t, want[k], got, "%s %v edge %d", fac, edges, i)
			}
		}
	}
}

func TestLotsOf_区域を同じ地区の中の区画で隙間なく埋める(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	ds := districtsFrom(raws)
	big := 0
	for _, d := range ds {
		for seed := range uint64(40) {
			r := d.regionIn(seed, 0, 7)
			covered := map[consts.Coord[consts.Chunk]]int{}
			for _, b := range lotsOf(raws, cat, r, 0) {
				if b.w*b.h > 1 {
					big++
				}
				z := r.zoneAt(b.anchor)
				for dy := range b.h {
					for dx := range b.w {
						c := b.anchor.Add(consts.Coord[consts.Chunk]{X: dx, Y: dy})
						covered[c]++
						assert.Equal(t, z, r.zoneAt(c), "区画は1つの地区に収まる")
					}
				}
			}
			assert.Len(t, covered, int(r.w*r.h), "区域の全てのチャンクを覆う")
			for c, n := range covered {
				assert.Equal(t, 1, n, "%v は1つの区画にだけ入る", c)
			}
		}
	}
	assert.Positive(t, big, "2チャンク以上の区画が出る")
}

func TestBuildingLot_roomEdges_外周は壁で入口と迷路の扉を開ける(t *testing.T) {
	t.Parallel()

	b := buildingLot{w: 3, h: 2, interior: rawschema.FacilityInteriorRooms, seed: 7}
	doors := 0
	for ly := range b.h {
		for lx := range b.w {
			l := consts.Coord[consts.Chunk]{X: lx, Y: ly}
			e := b.roomEdges(l)
			if lx == 0 {
				assert.Equal(t, maptemplate.EdgeWall, e[3])
			}
			if lx == b.w-1 {
				assert.Equal(t, maptemplate.EdgeWall, e[1])
			}
			if ly == b.h-1 {
				assert.Equal(t, maptemplate.EdgeWall, e[2])
			}
			if ly == 0 {
				want := maptemplate.EdgeWall
				if lx == b.w/2 {
					want = maptemplate.EdgeDoor
				}
				assert.Equal(t, want, e[0], "入口は北の辺の真ん中の列だけ")
			}
			// 東と南の内側の辺を数え、各辺を1回だけ数える
			if lx < b.w-1 && e[1] == maptemplate.EdgeDoor {
				doors++
			}
			if ly < b.h-1 && e[2] == maptemplate.EdgeDoor {
				doors++
			}
		}
	}
	assert.Equal(t, int(b.w*b.h)-1, doors, "迷路の扉は全域木の辺の数")

}

func TestBuildingLot_roomEdges_吹き抜けは外周が壁で内側の辺が全て吹き抜け(t *testing.T) {
	t.Parallel()

	const (
		W = maptemplate.EdgeWall
		D = maptemplate.EdgeDoor
		O = maptemplate.EdgeOpen
	)
	hall := buildingLot{w: 3, h: 2, interior: rawschema.FacilityInteriorHall, seed: 7}
	got := map[consts.Coord[consts.Chunk]][4]maptemplate.EdgeKind{}
	for ly := range hall.h {
		for lx := range hall.w {
			l := consts.Coord[consts.Chunk]{X: lx, Y: ly}
			got[l] = hall.roomEdges(l)
		}
	}
	assert.Equal(t, map[consts.Coord[consts.Chunk]][4]maptemplate.EdgeKind{
		{X: 0, Y: 0}: {W, O, O, W},
		{X: 1, Y: 0}: {D, O, O, O},
		{X: 2, Y: 0}: {W, W, O, O},
		{X: 0, Y: 1}: {O, O, W, W},
		{X: 1, Y: 1}: {O, O, W, O},
		{X: 2, Y: 1}: {O, W, W, O},
	}, got, "北・東・南・西の順。入口は北の辺の真ん中の列")
}

func TestGolden_Building(t *testing.T) {
	t.Parallel()

	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	for _, tt := range []struct {
		name string
		lot  buildingLot
	}{
		{"lab_main_3x3", buildingLot{facility: "lab_main", w: 3, h: 3, interior: rawschema.FacilityInteriorRooms, seed: 1}},
		{"lab_main_4x3", buildingLot{facility: "lab_main", w: 4, h: 3, interior: rawschema.FacilityInteriorRooms, seed: 2}},
		{"military_armory_4x1", buildingLot{facility: "military_armory", w: 4, h: 1, interior: rawschema.FacilityInteriorRooms, seed: 3}},
		{"power_turbine_3x3", buildingLot{facility: "power_turbine", w: 3, h: 3, interior: rawschema.FacilityInteriorHall, seed: 4}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cells := composeBuilding(t, testutil.InitTestWorld(t).Resources.RawMaster, loader, tt.lot)
			world := testutil.InitTestWorld(t)
			w, h := tt.lot.w.Tiles(testChunkSide), tt.lot.h.Tiles(testChunkSide)
			g := chunkGeom{chunkW: w, chunkH: h, tiles: &tileIndex{world: world, loX: 0, hiX: w}}
			_, _, err := spawnFacility(world, g, cells, 0, 0, tt.lot.facility, 1, tt.lot.seed)
			require.NoError(t, err)
			img, err := RenderRegion(world, image.Rect(0, 0, int(w), int(h)), RegionSpritePx)
			require.NoError(t, err)
			var buf bytes.Buffer
			require.NoError(t, png.Encode(&buf, img))
			goldie.New(t, goldie.WithNameSuffix(".png")).Assert(t, t.Name(), buf.Bytes())
		})
	}
}

func TestLotsOf_目玉の施設は区域に1つだけ地区の真ん中へ一番大きく建つ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	for _, d := range districtsFrom(raws) {
		if d.row.Landmark == nil {
			continue
		}
		for seed := range uint64(40) {
			r := d.regionIn(seed, 0, 7)
			var got []buildingLot
			for _, b := range lotsOf(raws, cat, r, 0) {
				if b.facility == *d.row.Landmark {
					got = append(got, b)
				}
			}
			require.Len(t, got, 1, "seed=%d", seed)
			assert.Equal(t, largestFittingFootprint(raws, r, *d.row.Landmark), got[0].w*got[0].h, "%s seed=%d 一番大きい候補から収まるものを選ぶ", d.row.Id, seed)
		}
	}
}

func TestFacilities_2チャンク以上になりうる施設は部屋の絵を持つ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	var want, got []string
	for _, f := range raw.PtrSlice(raws.Facilities) {
		for _, fp := range raw.PtrSlice(f.Footprints) {
			if fp.W*fp.H > 1 {
				want = append(want, f.Id)
				break
			}
		}
	}
	for _, id := range want {
		if loader.HasRooms(id) {
			got = append(got, id)
		}
	}
	assert.Equal(t, want, got)
}

// largestFittingFootprint は施設 fac の大きさの候補のうち、施設が建つ地区を囲む矩形に収まる一番大きい面積を返す
func largestFittingFootprint(raws rawschema.Raws, r districtRegion, fac string) consts.Chunk {
	f, _ := raw.GetFacility(raws, fac)
	minC, maxC := consts.Coord[consts.Chunk]{X: r.w, Y: r.h}, consts.Coord[consts.Chunk]{X: -1, Y: -1}
	for _, fz := range f.Zones {
		for y := range r.h {
			for x := range r.w {
				if r.zoneAt(consts.Coord[consts.Chunk]{X: x, Y: y}) == fz.Zone {
					minC = consts.Coord[consts.Chunk]{X: min(minC.X, x), Y: min(minC.Y, y)}
					maxC = consts.Coord[consts.Chunk]{X: max(maxC.X, x), Y: max(maxC.Y, y)}
				}
			}
		}
		if maxC.X >= 0 {
			break
		}
	}
	fps := facilityFootprints(f)
	var best consts.Chunk
	for _, fp := range fps {
		if consts.Chunk(fp.W) <= maxC.X-minC.X+1 && consts.Chunk(fp.H) <= maxC.Y-minC.Y+1 {
			best = max(best, consts.Chunk(fp.W*fp.H))
		}
	}
	return best
}

// possibleRoomEdges は区画 b の中の座標 l のチャンクが、seed によらず取りうる北・東・南・西の辺の組を全て返す。
// 区画の外へ向く辺は壁で、北の辺の真ん中の列だけ入口の扉になる。内側の辺は吹き抜けなら吹き抜け、迷路なら壁か扉で、
// 迷路は全域木なので全ての部屋が扉を1つ以上持つ
func possibleRoomEdges(b buildingLot, l consts.Coord[consts.Chunk]) [][4]maptemplate.EdgeKind {
	dirs := [4]consts.Coord[consts.Chunk]{{Y: -1}, {X: 1}, {Y: 1}, {X: -1}}
	var choices [4][]maptemplate.EdgeKind
	for i, d := range dirs {
		n := l.Add(d)
		switch {
		case i == 0 && l.Y == 0 && l.X == b.w/2:
			choices[i] = []maptemplate.EdgeKind{maptemplate.EdgeDoor}
		case n.X < 0 || n.Y < 0 || n.X >= b.w || n.Y >= b.h:
			choices[i] = []maptemplate.EdgeKind{maptemplate.EdgeWall}
		case b.interior == rawschema.FacilityInteriorHall:
			choices[i] = []maptemplate.EdgeKind{maptemplate.EdgeOpen}
		default:
			choices[i] = []maptemplate.EdgeKind{maptemplate.EdgeWall, maptemplate.EdgeDoor}
		}
	}
	var out [][4]maptemplate.EdgeKind
	for _, n := range choices[0] {
		for _, e := range choices[1] {
			for _, so := range choices[2] {
				for _, w := range choices[3] {
					edges := [4]maptemplate.EdgeKind{n, e, so, w}
					if b.interior == rawschema.FacilityInteriorRooms && !slices.Contains(edges[:], maptemplate.EdgeDoor) {
						continue
					}
					out = append(out, edges)
				}
			}
		}
	}
	return out
}

func TestRoomWall_部屋の絵の高さのある地形を帯の壁にする(t *testing.T) {
	t.Parallel()
	isTall := func(terrain string) bool { return terrain == "wall_brick_1" }
	brick := [][]maptemplate.MapCell{{{Terrain: "floor"}, {Terrain: "wall_brick_1"}}}
	open := [][]maptemplate.MapCell{{{Terrain: "floor"}, {Terrain: "floor"}}}
	assert.Equal(t, "wall_brick_1", roomWall(brick, isTall))
	assert.Equal(t, consts.TileNameWall, roomWall(open, isTall))
}

// TestFacilityTemplates_施設の絵は高さのある地形を1種類だけ使う は、施設の壁を1つの素材でそろえる前提を固定する。
// 部屋の絵に別の地形が混ざると帯の壁が走査の順で決まる。窓は扉と同じく床の上に置くので、高さのある地形に数えない。
// 部屋の絵で組む施設は帯の壁に使うので必ず1種類持ち、屋外の施設は壁を持たなくてよい
func TestFacilityTemplates_施設の絵は高さのある地形を1種類だけ使う(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	isTall := tileIsTall(raws)
	for _, fac := range loader.Facilities() {
		tall := map[string]bool{}
		for _, name := range loader.FacilityChunkNames(fac) {
			chunks, err := loader.GetChunks(name)
			require.NoError(t, err)
			for _, c := range chunks {
				cells, err := c.ExpandWithPlacements(loader, 1)
				require.NoError(t, err, name)
				for _, row := range cells {
					for _, cell := range row {
						if isTall(cell.Terrain) {
							tall[cell.Terrain] = true
						}
					}
				}
			}
		}
		if loader.HasRooms(fac) {
			assert.Len(t, tall, 1, "%s の絵の高さのある地形 %v", fac, slices.Collect(maps.Keys(tall)))
		} else {
			assert.LessOrEqual(t, len(tall), 1, "%s の絵の高さのある地形 %v", fac, slices.Collect(maps.Keys(tall)))
		}
	}
}
