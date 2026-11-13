package overworld

import (
	"bytes"
	"fmt"
	"image/png"
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

// composeBuilding は区画 b の全てのチャンクのマスを組んで、区画全体の1枚のマスにする
func composeBuilding(t *testing.T, loader *maptemplate.TemplateLoader, b buildingLot) [][]maptemplate.MapCell {
	t.Helper()
	side := int(testChunkSide)
	cells := make([][]maptemplate.MapCell, int(b.h)*side)
	for y := range cells {
		cells[y] = make([]maptemplate.MapCell, int(b.w)*side)
	}
	for ly := range b.h {
		for lx := range b.w {
			c, _, err := roomChunkCells(loader, b, consts.Coord[consts.Chunk]{X: lx, Y: ly}, testChunkSide, testChunkSide)
			require.NoError(t, err)
			for y, row := range c {
				copy(cells[int(ly)*side+y][int(lx)*side:], row)
			}
		}
	}
	return cells
}

func TestBuilding_研究所の本館は入口から全ての部屋へ入れる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	blockedProp, blockedTile := map[string]bool{}, map[string]bool{}
	for _, p := range raw.PtrSlice(raws.Props) {
		blockedProp[p.Id] = p.BlockPass
	}
	for _, tl := range raw.PtrSlice(raws.Tiles) {
		blockedTile[tl.Id] = tl.BlockPass
	}
	storage := map[string]bool{}
	for _, p := range raw.PtrSlice(raws.Props) {
		storage[p.Id] = p.Storage != nil
	}

	for w := consts.Chunk(1); w <= 4; w++ {
		for h := consts.Chunk(1); h <= 4; h++ {
			for seed := range uint64(30) {
				b := buildingLot{facility: "lab_main", w: w, h: h, interior: rawschema.FacilityInteriorRooms, seed: seed}
				cells := composeBuilding(t, loader, b)
				name := fmt.Sprintf("%dx%d seed=%d", w, h, seed)
				assert.Empty(t, unreachableCells(cells, blockedTile, blockedProp), name)
				for y, row := range cells {
					for x, c := range row {
						if storage[c.Prop] {
							assert.NotEmpty(t, c.LootTable, "%s %s at %d,%d", name, c.Prop, x, y)
						}
					}
				}
			}
		}
	}
}

func TestFacilityRooms_扉の辺は真ん中の2マスが扉で壁の辺は全て壁(t *testing.T) {
	t.Parallel()

	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	kinds := []maptemplate.EdgeKind{maptemplate.EdgeWall, maptemplate.EdgeDoor}
	for _, fac := range loader.Facilities() {
		if !loader.HasRooms(fac) {
			continue
		}
		for _, n := range kinds {
			for _, e := range kinds {
				for _, s := range kinds {
					for _, wk := range kinds {
						edges := [4]maptemplate.EdgeKind{n, e, s, wk}
						if edges == [4]maptemplate.EdgeKind{maptemplate.EdgeWall, maptemplate.EdgeWall, maptemplate.EdgeWall, maptemplate.EdgeWall} {
							continue
						}
						_, cells, err := loader.FacilityRoom(fac, edges, 1)
						require.NoError(t, err, "%s %v", fac, edges)
						last := len(cells) - 1
						line := func(i int) [20]maptemplate.MapCell {
							var out [20]maptemplate.MapCell
							for j := range 20 {
								switch i {
								case 0:
									out[j] = cells[0][j]
								case 1:
									out[j] = cells[j][last]
								case 2:
									out[j] = cells[last][j]
								default:
									out[j] = cells[j][0]
								}
							}
							return out
						}
						for i, k := range edges {
							got := ""
							for _, c := range line(i) {
								switch {
								case c.Prop == "door":
									got += "+"
								case c.Terrain == "wall":
									got += "#"
								default:
									got += "."
								}
							}
							want := "####################"
							if k == maptemplate.EdgeDoor {
								want = "#########++#########"
							}
							assert.Equal(t, want, got, "%s %v edge %d", fac, edges, i)
						}
					}
				}
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
			for _, b := range lotsOf(raws, cat, r) {
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

	hall := buildingLot{w: 2, h: 2, interior: rawschema.FacilityInteriorHall, seed: 7}
	assert.Equal(t, [4]maptemplate.EdgeKind{maptemplate.EdgeOpen, maptemplate.EdgeOpen, maptemplate.EdgeWall, maptemplate.EdgeWall}, hall.roomEdges(consts.Coord[consts.Chunk]{X: 0, Y: 1}))
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cells := composeBuilding(t, loader, tt.lot)
			world := testutil.InitTestWorld(t)
			w, h := tt.lot.w.Tiles(testChunkSide), tt.lot.h.Tiles(testChunkSide)
			g := chunkGeom{chunkW: w, chunkH: h, tiles: &tileIndex{world: world, loX: 0, hiX: w}}
			_, _, err := spawnFacility(world, g, cells, 0, 0, tt.lot.facility, 1, tt.lot.seed, nil)
			require.NoError(t, err)
			img := renderOverworldRegion(t, world, 0, 0, int(w), int(h))
			var buf bytes.Buffer
			require.NoError(t, png.Encode(&buf, img))
			goldie.New(t, goldie.WithNameSuffix(".png")).Assert(t, t.Name(), buf.Bytes())
		})
	}
}
