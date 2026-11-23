package overworld

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFacilityTemplates_扉が塞がらず全ての床へ入れる は施設の絵を seed と向きを変えて展開し、扉の両側が通れて、
// 通れる全てのマスへ区画の縁から歩いて入れることを確かめる。部品が扉の前に家具を置く組み合わせもここで落とす
func TestFacilityTemplates_扉が塞がらず全ての床へ入れる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)

	rules := newCellRules(raws)

	for _, fac := range loader.Facilities() {
		// 部屋の絵を組む建物は building_test.go で確かめる
		if loader.HasRooms(fac) {
			continue
		}
		for seed := range uint64(200) {
			_, base, err := loader.FacilityChunk(fac, seed)
			require.NoError(t, err)
			for turns := range 4 {
				for _, mirror := range []bool{false, true} {
					cells := maptemplate.TransformCells(base, turns, mirror)
					name := fmt.Sprintf("%s seed=%d turns=%d mirror=%v", fac, seed, turns, mirror)
					assert.Empty(t, unreachableCells(cells, rules), name)
				}
			}
		}
	}
}

// cellRules は施設の絵のマスが通れるかを決める、タイルと置物の行の性質
type cellRules struct {
	blockedTile, blockedProp, door, storage map[string]bool
}

// newCellRules は raw の行から、通れないタイルと置物、扉の置物を集める
func newCellRules(raws rawschema.Raws) cellRules {
	r := cellRules{blockedTile: map[string]bool{}, blockedProp: map[string]bool{}, door: map[string]bool{}, storage: map[string]bool{}}
	tiles, props := raw.PtrSlice(raws.Tiles), raw.PtrSlice(raws.Props)
	for i := range tiles {
		r.blockedTile[tiles[i].Id] = tiles[i].BlockPass
	}
	for i := range props {
		r.blockedProp[props[i].Id] = props[i].BlockPass
		r.door[props[i].Id] = props[i].Door != nil
		r.storage[props[i].Id] = props[i].Storage != nil
	}
	return r
}

// unreachableCells は扉の片側が塞がっている扉と、区画の縁から歩いて入れない置物の無い通れるマスと、
// 歩いて入れるマスから探せない収納を返す
func unreachableCells(cells [][]maptemplate.MapCell, rules cellRules) []string {
	h, w := len(cells), len(cells[0])
	passable := func(x, y int) bool {
		if x < 0 || y < 0 || x >= w || y >= h {
			return false
		}
		c := cells[y][x]
		if rules.blockedTile[c.Terrain] {
			return false
		}
		return c.Prop == "" || rules.door[c.Prop] || !rules.blockedProp[c.Prop]
	}

	bad := blockedDoors(cells, rules.door, passable)

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
			// 家具のマスは通り道でなく囲まれていてもよい。収納は下で探せるかを確かめる
			if passable(x, y) && cells[y][x].Prop == "" && !seen[[2]int{x, y}] {
				bad = append(bad, fmt.Sprintf("unreachable %s at %d,%d", cells[y][x].Terrain, x, y))
			}
		}
	}
	return append(bad, unsearchableStorage(cells, rules.storage, seen)...)
}

// unsearchableStorage は、歩いて入れるマス seen の上にも8方向の隣にも無い収納を返す。収納は同じマスか隣から探す
func unsearchableStorage(cells [][]maptemplate.MapCell, storage map[string]bool, seen map[[2]int]bool) []string {
	var bad []string
	for y := range cells {
		for x := range cells[y] {
			if !storage[cells[y][x].Prop] {
				continue
			}
			searchable := false
			for dy := -1; dy <= 1 && !searchable; dy++ {
				for dx := -1; dx <= 1 && !searchable; dx++ {
					searchable = seen[[2]int{x + dx, y + dy}]
				}
			}
			if !searchable {
				bad = append(bad, fmt.Sprintf("unsearchable %s at %d,%d", cells[y][x].Prop, x, y))
			}
		}
	}
	return bad
}

// blockedDoors は両側の向かい合うマスが通れない扉を返す
func blockedDoors(cells [][]maptemplate.MapCell, door map[string]bool, passable func(x, y int) bool) []string {
	var bad []string
	for y := range cells {
		for x := range cells[y] {
			if !door[cells[y][x].Prop] {
				continue
			}
			throughX := passable(x-1, y) && passable(x+1, y)
			throughY := passable(x, y-1) && passable(x, y+1)
			if !throughX && !throughY {
				bad = append(bad, fmt.Sprintf("door blocked at %d,%d", x, y))
			}
		}
	}
	return bad
}

func TestFacilityTemplates_収納家具は戦利品表を持つ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)

	storage := map[string]bool{}
	for _, p := range raw.PtrSlice(raws.Props) {
		storage[p.Id] = p.Storage != nil
	}
	for _, fac := range loader.Facilities() {
		// 部屋の絵を組む建物は building_test.go で確かめる
		if loader.HasRooms(fac) {
			continue
		}
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
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)

	facilities := raw.PtrSlice(raws.Facilities)
	want := make([]string, 0, len(facilities))
	for _, f := range facilities {
		want = append(want, f.Id)
	}
	assert.ElementsMatch(t, want, loader.Facilities())
}

func TestSpawnFacility_窓に挟まれた扉は縦向きになる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	cells := [][]maptemplate.MapCell{{
		{Terrain: "floor_linoleum", Prop: "window"},
		{Terrain: "floor_linoleum", Prop: "door"},
		{Terrain: "floor_linoleum", Prop: "window"},
	}}
	g := chunkGeom{chunkW: 3, chunkH: 1, tiles: &tileIndex{world: world, loX: 0, hiX: 3}}
	_, _, err := spawnFacility(world, g, cells, 0, 0, "store", raw.MinDanger, 1)
	require.NoError(t, err)

	var got []gc.DoorOrientation
	q := ecs.NewFilter1[gc.Door](world.ECS).Query()
	for q.Next() {
		got = append(got, q.Get().Orientation)
	}
	assert.Equal(t, []gc.DoorOrientation{gc.DoorOrientationVertical}, got)
}

func TestSpawnFacility_棚の向きを施設の絵の並びから決めて付ける(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	cells := [][]maptemplate.MapCell{
		{{Terrain: "wall"}, {Terrain: "wall"}},
		{{Terrain: "floor_linoleum", Prop: "book_shelf_2"}, {Terrain: "floor_linoleum", Prop: "book_shelf_2"}},
	}
	g := chunkGeom{chunkW: 2, chunkH: 2, tiles: &tileIndex{world: world, loX: 0, hiX: 2}}
	_, _, err := spawnFacility(world, g, cells, 0, 0, "store", raw.MinDanger, 1)
	require.NoError(t, err)

	var got []gc.Pose
	q := ecs.NewFilter1[gc.Pose](world.ECS).Query()
	for q.Next() {
		got = append(got, *q.Get())
	}
	assert.Equal(t, []gc.Pose{{Facing: consts.South, Anchor: gc.AnchorBack}, {Facing: consts.South, Anchor: gc.AnchorBack}}, got)
}

func TestSpawnFacility_扉の向きを決める隣の置物の行が無ければエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	cells := [][]maptemplate.MapCell{{
		{Terrain: "floor_linoleum", Prop: "door"},
		{Terrain: "floor_linoleum", Prop: "no_such_prop"},
	}}
	g := chunkGeom{chunkW: 2, chunkH: 1, tiles: &tileIndex{world: world, loX: 0, hiX: 2}}
	_, _, err := spawnFacility(world, g, cells, 0, 0, "store", raw.MinDanger, 1)
	var notFound raw.KeyNotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, raw.KeyNotFoundError{Key: "no_such_prop", Collection: "Props"}, notFound)
	// 置物を置く前に、扉の向きを決める時点で止まるので、先に並べた扉も置かれない
	doors := 0
	q := ecs.NewFilter1[gc.Door](world.ECS).Query()
	for q.Next() {
		doors++
	}
	assert.Equal(t, 0, doors)
}

func TestSpawnFacility_窓は通れず視線を通し下の床も塞がない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	cells := [][]maptemplate.MapCell{{{Terrain: "floor_linoleum", Prop: "window"}}}
	g := chunkGeom{chunkW: 1, chunkH: 1, tiles: &tileIndex{world: world, loX: 0, hiX: 1}}
	_, _, err := spawnFacility(world, g, cells, 0, 0, "store", raw.MinDanger, 1)
	require.NoError(t, err)

	type traits struct{ pass, view bool }
	got := map[string]traits{}
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		kind := "tile"
		if world.Components.Prop.Has(e) {
			kind = "prop"
		}
		got[kind] = traits{pass: world.Components.BlockPass.Has(e), view: world.Components.BlockView.Has(e)}
	}
	assert.Equal(t, map[string]traits{
		"prop": {pass: true, view: false},
		"tile": {pass: false, view: false},
	}, got)
}

func TestSpawnFacility_畳は屋内で庭は下地のまま屋外(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	for y := range testChunkSide - urbanStreetW {
		for x := range testChunkSide - urbanStreetW {
			_, err := lifecycle.SpawnTile(world, "dirt_tundra", x, y, new(15))
			require.NoError(t, err)
		}
	}
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	_, cells, err := loader.FacilityChunk("house", 1)
	require.NoError(t, err)
	lot := testChunkSide - urbanStreetW
	g := chunkGeom{chunkW: lot, chunkH: lot, tiles: &tileIndex{world: world, loX: 0, hiX: lot}}
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
	for _, id := range []string{"floor_tatami", "dirt_tundra"} {
		require.NotEmpty(t, got[id], id)
	}
	assert.Equal(t, []rawschema.Shelter{rawschema.Indoor}, slices.Collect(maps.Keys(got["floor_tatami"])))
	assert.Equal(t, []rawschema.Shelter{rawschema.Outdoor}, slices.Collect(maps.Keys(got["dirt_tundra"])), "庭は置かれた場所の下地のまま")
}

func TestVoidEnclosedTall_通れるマスに接しない高さのある地形だけを虚空にする(t *testing.T) {
	t.Parallel()

	parse := func(rows []string) [][]maptemplate.MapCell {
		cells := make([][]maptemplate.MapCell, len(rows))
		for y, row := range rows {
			cells[y] = make([]maptemplate.MapCell, len(row))
			for x, ch := range row {
				switch ch {
				case '#':
					cells[y][x] = maptemplate.MapCell{Terrain: consts.TileNameWall}
				case 'w':
					cells[y][x] = maptemplate.MapCell{Terrain: consts.TileNameWall, Prop: "window"}
				default:
					cells[y][x] = maptemplate.MapCell{Terrain: consts.TileNameFloor}
				}
			}
		}
		return cells
	}
	render := func(cells [][]maptemplate.MapCell) []string {
		out := make([]string, len(cells))
		for y, row := range cells {
			for _, c := range row {
				switch {
				case c.Terrain == consts.TileNameVoid:
					out[y] += " "
				case c.Prop != "":
					out[y] += "w"
				case c.Terrain == consts.TileNameWall:
					out[y] += "#"
				default:
					out[y] += "."
				}
			}
		}
		return out
	}
	blocks := func(terrain string) bool { return terrain == consts.TileNameWall || terrain == consts.TileNameVoid }
	isTall := func(terrain string) bool { return terrain == consts.TileNameWall }
	rows := []string{
		"######",
		"######",
		"##w###",
		"######",
		"#....#",
	}

	walled := parse(rows)
	voidEnclosedTall(walled, blocks, isTall, func(int, int) bool { return false })
	assert.Equal(t, []string{
		"      ",
		"      ",
		"  w   ",
		"######",
		"#....#",
	}, render(walled), "マスの外を壁と数えると、外周の壁も床に接しなければ虚空になる")

	open := parse(rows)
	voidEnclosedTall(open, blocks, isTall, func(int, int) bool { return true })
	assert.Equal(t, []string{
		"######",
		"#    #",
		"# w  #",
		"######",
		"#....#",
	}, render(open), "マスの外を通れると数えると、外周の壁は残る")

	east := parse(rows)
	voidEnclosedTall(east, blocks, isTall, func(x, _ int) bool { return x >= 6 })
	assert.Equal(t, []string{
		"     #",
		"     #",
		"  w  #",
		"######",
		"#....#",
	}, render(east), "通れる外に面する辺の壁だけが残る")

	// 高さのない通れない地形は、囲まれていても虚空にしない
	water := [][]maptemplate.MapCell{
		{{Terrain: "deep_water"}, {Terrain: "deep_water"}, {Terrain: "deep_water"}},
		{{Terrain: "deep_water"}, {Terrain: "deep_water"}, {Terrain: "deep_water"}},
		{{Terrain: "deep_water"}, {Terrain: "deep_water"}, {Terrain: "deep_water"}},
	}
	voidEnclosedTall(water, func(string) bool { return true }, isTall, func(int, int) bool { return false })
	assert.Equal(t, "deep_water", water[1][1].Terrain, "囲まれた水は水のまま")
}

// TestFacilityTemplates_置物の下の地形は置物より塞がない は、壊したら下の地形が現れる置物の前提を固定する。
// 通行を塞ぐ置物の下は通れ、視線を通す置物の下は視線を通す。下の地形が塞ぐと、置物の性質が打ち消される
func TestFacilityTemplates_置物の下の地形は置物より塞がない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	tiles := map[string]rawschema.Tile{}
	for _, tl := range raw.PtrSlice(raws.Tiles) {
		tiles[tl.Id] = tl
	}
	props := map[string]rawschema.Prop{}
	for _, p := range raw.PtrSlice(raws.Props) {
		props[p.Id] = p
	}

	var bad []string
	for _, fac := range loader.Facilities() {
		for _, name := range loader.FacilityChunkNames(fac) {
			_, _, cells, err := loader.LoadTemplateByName(name, 1)
			require.NoError(t, err)
			for y, row := range cells {
				for x, c := range row {
					p, ok := props[c.Prop]
					if !ok {
						continue
					}
					tl := tiles[c.Terrain]
					if (p.BlockPass && tl.BlockPass) || (!p.BlockView && tl.BlockView) {
						bad = append(bad, fmt.Sprintf("%s %d,%d %s on %s", name, x, y, c.Prop, c.Terrain))
					}
				}
			}
		}
	}
	assert.Empty(t, bad)
}

// TestFacilityTemplates_囲いの外の床は屋外で内は屋内 は1棟の絵で、区画の縁から壁と通行を塞ぐ置物を越えずにたどれる床が屋外のタイルを、
// たどれない床が屋内のタイルを指すことを確かめる。部屋の絵は床が全て屋内のタイルを指すことを確かめる。屋内外はタイルの種類で決まるので、絵が置き場所に合うタイルを選ぶ必要がある
func TestFacilityTemplates_囲いの外の床は屋外で内は屋内(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)

	rows := map[string]rawschema.Tile{}
	for _, tl := range raw.PtrSlice(raws.Tiles) {
		rows[tl.Id] = tl
	}
	// 扉や窓のように通行を塞ぐ置物は、壁と同じく囲いになる
	propBlocks := map[string]bool{}
	for _, p := range raw.PtrSlice(raws.Props) {
		propBlocks[p.Id] = p.BlockPass
	}

	for _, fac := range loader.Facilities() {
		for _, name := range loader.FacilityChunkNames(fac) {
			for seed := range uint64(20) {
				chunk, _, cells, err := loader.LoadTemplateByName(name, seed)
				require.NoError(t, err)
				// 部屋の絵の縁の先は隣の部屋か建物の中の帯なので、部屋の絵の床は全て屋内
				outside := map[[2]int]bool{}
				if chunk.Edges == nil {
					outside = reachableFromEdge(cells, func(x, y int) bool {
						return rows[cells[y][x].Terrain].BlockPass || propBlocks[cells[y][x].Prop]
					})
				}
				var bad []string
				for y, row := range cells {
					for x, c := range row {
						if rows[c.Terrain].BlockPass || propBlocks[c.Prop] {
							continue
						}
						want := rawschema.Indoor
						if outside[[2]int{x, y}] {
							want = rawschema.Outdoor
						}
						if got := raw.TileShelter(rows[c.Terrain]); got != want {
							bad = append(bad, fmt.Sprintf("%d,%d %s is %s, want %s", x, y, c.Terrain, got, want))
						}
					}
				}
				assert.Empty(t, bad, "%s seed=%d", name, seed)
			}
		}
	}
}

// reachableFromEdge は縁のマスから、blocks のマスを越えずに4方向でたどれるマスを返す
func reachableFromEdge(cells [][]maptemplate.MapCell, blocks func(x, y int) bool) map[[2]int]bool {
	h, w := len(cells), len(cells[0])
	seen := map[[2]int]bool{}
	var queue [][2]int
	for y := range h {
		for x := range w {
			if (x == 0 || y == 0 || x == w-1 || y == h-1) && !blocks(x, y) {
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
			if n[0] < 0 || n[1] < 0 || n[0] >= w || n[1] >= h || seen[n] || blocks(n[0], n[1]) {
				continue
			}
			seen[n] = true
			queue = append(queue, n)
		}
	}
	return seen
}
