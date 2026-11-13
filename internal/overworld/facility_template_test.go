package overworld

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFacilityTemplates_扉が塞がらず全ての床へ入れる は施設の絵を seed と向きを変えて展開し、扉の両側が通れて、
// 通れる全てのマスへ区画の縁から歩いて入れることを確かめる。部品が扉の前に家具を置く組み合わせもここで落とす
func TestFacilityTemplates_扉が塞がらず全ての床へ入れる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := facilityTemplates()
	require.NoError(t, err)

	blockedProp := map[string]bool{}
	for _, p := range raw.PtrSlice(raws.Props) {
		blockedProp[p.Id] = p.BlockPass
	}
	blockedTile := map[string]bool{}
	for _, tl := range raw.PtrSlice(raws.Tiles) {
		blockedTile[tl.Id] = tl.BlockPass
	}

	for _, fac := range loader.Facilities() {
		for seed := range uint64(200) {
			_, base, err := loader.FacilityChunk(fac, seed)
			require.NoError(t, err)
			for turns := range 4 {
				for _, mirror := range []bool{false, true} {
					cells := maptemplate.TransformCells(base, turns, mirror)
					name := fmt.Sprintf("%s seed=%d turns=%d mirror=%v", fac, seed, turns, mirror)
					assert.Empty(t, unreachableCells(cells, blockedTile, blockedProp), name)
				}
			}
		}
	}
}

// unreachableCells は扉の片側が塞がっている扉と、区画の縁から歩いて入れない通れるマスを返す
func unreachableCells(cells [][]maptemplate.MapCell, blockedTile, blockedProp map[string]bool) []string {
	h, w := len(cells), len(cells[0])
	passable := func(x, y int) bool {
		if x < 0 || y < 0 || x >= w || y >= h {
			return false
		}
		c := cells[y][x]
		if blockedTile[c.Terrain] {
			return false
		}
		return c.Prop == "" || c.Prop == "door" || !blockedProp[c.Prop]
	}

	var bad []string
	for y := range h {
		for x := range w {
			if cells[y][x].Prop != "door" {
				continue
			}
			throughX := passable(x-1, y) && passable(x+1, y)
			throughY := passable(x, y-1) && passable(x, y+1)
			if !throughX && !throughY {
				bad = append(bad, fmt.Sprintf("door blocked at %d,%d", x, y))
			}
		}
	}

	seen := map[[2]int]bool{}
	var queue [][2]int
	for y := range h {
		for x := range w {
			if (x == 0 || y == 0 || x == w-1 || y == h-1) && passable(x, y) {
				seen[[2]int{x, y}] = true
				queue = append(queue, [2]int{x, y})
			}
		}
	}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := [2]int{c[0] + d[0], c[1] + d[1]}
			if seen[n] || !passable(n[0], n[1]) {
				continue
			}
			seen[n] = true
			queue = append(queue, n)
		}
	}
	for y := range h {
		for x := range w {
			if passable(x, y) && !seen[[2]int{x, y}] {
				bad = append(bad, fmt.Sprintf("unreachable %s at %d,%d", cells[y][x].Terrain, x, y))
			}
		}
	}
	return bad
}

func TestFacilityTemplates_収納家具は戦利品表を持つ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := facilityTemplates()
	require.NoError(t, err)

	storage := map[string]bool{}
	for _, p := range raw.PtrSlice(raws.Props) {
		storage[p.Id] = p.Storage != nil
	}
	for _, fac := range loader.Facilities() {
		for seed := range uint64(200) {
			_, cells, err := loader.FacilityChunk(fac, seed)
			require.NoError(t, err)
			for y, row := range cells {
				for x, c := range row {
					if storage[c.Prop] {
						assert.NotEmpty(t, c.LootTable, "%s seed=%d %s at %d,%d", fac, seed, c.Prop, x, y)
					}
				}
			}
		}
	}
}

func TestFacilityTemplates_全施設に絵がある(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := facilityTemplates()
	require.NoError(t, err)

	facilities := raw.PtrSlice(raws.Facilities)
	want := make([]string, 0, len(facilities))
	for _, f := range facilities {
		want = append(want, f.Id)
	}
	assert.ElementsMatch(t, want, loader.Facilities())
}

func TestSpawnFacility_畳は屋内で土は屋外(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	loader, err := facilityTemplates()
	require.NoError(t, err)
	_, cells, err := loader.FacilityChunk("house", 1)
	require.NoError(t, err)
	g := chunkGeom{chunkW: facilityLot, chunkH: facilityLot, tiles: &tileIndex{world: world, loX: 0, hiX: facilityLot}}
	_, _, err = spawnFacility(world, g, cells, 0, 0, "house", raw.MinDanger, 1)
	require.NoError(t, err)

	got := map[string]map[rawschema.Shelter]int{}
	for _, e := range g.tiles.get() {
		if !world.Components.TileEnvironment.Has(e) {
			continue
		}
		id := world.Components.RawID.Get(e).ID
		if got[id] == nil {
			got[id] = map[rawschema.Shelter]int{}
		}
		got[id][world.Components.TileEnvironment.Get(e).Shelter]++
	}
	for _, id := range []string{"floor_tatami", consts.TileNameDirt} {
		require.NotEmpty(t, got[id], id)
	}
	assert.Equal(t, []rawschema.Shelter{rawschema.Indoor}, slices.Collect(maps.Keys(got["floor_tatami"])))
	assert.Equal(t, []rawschema.Shelter{rawschema.Outdoor}, slices.Collect(maps.Keys(got[consts.TileNameDirt])))
}
