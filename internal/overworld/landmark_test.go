package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findLandmarkChunk は点在ランドマークが当選し、他の地物に譲らないチャンクと seed を探す。
func findLandmarkChunk(t *testing.T) (uint64, consts.Coord[consts.Chunk]) {
	t.Helper()
	const rows = 1
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	for s := uint64(1); s < 200; s++ {
		for y := range consts.Chunk(30) {
			c := consts.Coord[consts.Chunk]{Y: y}
			// 帯を作らずに置くので、生成の側は開始の行を0とみなし、チャンク (0,0) は墜落現場になる
			if chunkTypeAt(raws, s, c, rows, 0) != chunkLandmark {
				continue
			}
			return s, c
		}
	}
	require.Fail(t, "前提: ランドマークだけが当選するチャンクが見つかる")
	return 0, consts.Coord[consts.Chunk]{}
}

// landmarkEntities は名前を持つエンティティを 座標 → 名前 の対応で集める。座標を文字列へ畳まず
// consts.Coord をそのまま map のキーにするので、桁や区切りの表記に依存せず整列も要らない。
func landmarkEntities(world w.World) map[consts.Coord[consts.Tile]]string {
	got := map[consts.Coord[consts.Tile]]string{}
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		got[world.Components.GridElement.Get(e).Coord] = string(world.Components.Name.Get(e).Name)
	}
	return got
}

func TestWildernessLandmark_原野の当選チャンクに小構造物が決定的に出る(t *testing.T) {
	t.Parallel()

	seed, c := findLandmarkChunk(t)

	build := func() map[consts.Coord[consts.Tile]]string {
		world := testutil.InitTestWorld(t)
		for y := range consts.Tile(50) {
			for x := range consts.Tile(50) {
				_, err := lifecycle.SpawnTile(world, "dirt_tundra", x, y, new(15))
				require.NoError(t, err)
			}
		}
		g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: 50, chunkH: 50, tiles: &tileIndex{world: world, loX: 0, hiX: 50}}
		require.NoError(t, wildernessLandmarkFeature{}.place(world, seed, c, 1, g))
		return landmarkEntities(world)
	}
	a := build()
	b := build()
	assert.NotEmpty(t, a, "ランドマークの prop が置かれる")
	assert.Equal(t, a, b, "ランドマークの配置は決定的で再生成しても一致する")
}

func TestLandmarks_全てのランドマークの施設が施設の絵を持つ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	for _, lm := range raw.PtrSlice(raws.Landmarks) {
		_, _, err := loader.FacilityChunk(lm.Id, 1)
		require.NoError(t, err, lm.Id)
	}
}

func TestLandmarks_全ての絵が余白を空けてチャンクに収まる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	chunkW, chunkH, _, _ := dungeon.DungeonOverworld.BandShape()
	// 絵は回転して置くので、幅と高さの大きい方が短い辺に収まる必要がある
	limit := int(min(chunkW, chunkH) - 2*landmarkMargin)
	for _, lm := range raw.PtrSlice(raws.Landmarks) {
		names := loader.FacilityChunkNames(lm.Id)
		require.NotEmpty(t, names, lm.Id)
		for _, name := range names {
			_, _, cells, err := loader.LoadTemplateByName(name, 1)
			require.NoError(t, err, name)
			assert.LessOrEqual(t, max(len(cells), len(cells[0])), limit, name)
		}
	}
}

func TestSpawnFacility_絵がチャンクからはみ出すとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	g := chunkGeom{chunkW: 8, chunkH: 8, tiles: &tileIndex{world: world, loX: 0, hiX: 8}}
	cells := make([][]maptemplate.MapCell, 5)
	for y := range cells {
		cells[y] = make([]maptemplate.MapCell, 5)
		for x := range cells[y] {
			cells[y][x] = maptemplate.MapCell{Terrain: consts.TileNameDirt}
		}
	}
	_, _, err := spawnFacility(world, g, cells, 3, 3, "campsite", 1, 1)
	require.NoError(t, err, "右下の端までちょうど収まる")
	_, _, err = spawnFacility(world, g, cells, 4, 3, "campsite", 1, 1)
	require.ErrorIs(t, err, errFacilityOutOfChunk)
}

func TestSpawnFacility_下地を残すマスの下が土の地面でないとエラー(t *testing.T) {
	t.Parallel()

	cells := [][]maptemplate.MapCell{{{Terrain: maptemplate.GroundTerrain}}}
	tests := []struct {
		name    string
		under   string
		wantErr error
	}{
		{"土の地面なら置ける", "dirt_tundra", nil},
		{"床の上はエラー", "floor_tile", errFacilityGroundNotEarth},
		{"タイルが無いとエラー", "", errFacilityGroundNotEarth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			if tt.under != "" {
				_, err := lifecycle.SpawnTile(world, tt.under, 0, 0, nil)
				require.NoError(t, err)
			}
			g := chunkGeom{chunkW: 1, chunkH: 1, tiles: &tileIndex{world: world, loX: 0, hiX: 1}}
			_, _, err := spawnFacility(world, g, cells, 0, 0, "campsite", 1, 1)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestLandmarkOrigins_道から間を空けた位置だけを返す(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 20, 20
	roads := map[consts.Coord[consts.Tile]]bool{}
	for x := range chunkW {
		roads[consts.Coord[consts.Tile]{X: x, Y: 6}] = true
	}

	got := landmarkOrigins(5, 4, chunkW, chunkH, roads)

	// 縁から1マス、道の6行目からも1マス空けるので、原点の行は北側の1と南側の8から15、列は1から14になる
	var want []consts.Coord[consts.Tile]
	for _, y := range []consts.Tile{1, 8, 9, 10, 11, 12, 13, 14, 15} {
		for x := consts.Tile(1); x <= 14; x++ {
			want = append(want, consts.Coord[consts.Tile]{X: x, Y: y})
		}
	}
	assert.Equal(t, want, got)
}

func TestLandmarkOrigins_道で埋まれば置き場所が無い(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 10, 10
	roads := map[consts.Coord[consts.Tile]]bool{}
	for x := range chunkW {
		roads[consts.Coord[consts.Tile]{X: x, Y: 5}] = true
	}

	assert.Empty(t, landmarkOrigins(5, 4, chunkW, chunkH, roads))
}

func TestWildernessLandmark_道の通るチャンクでも道を塞がない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	chunkW, chunkH, cols, _ := dungeon.DungeonOverworld.BandShape()
	type found struct {
		seed  uint64
		c     consts.Coord[consts.Chunk]
		roads map[consts.Coord[consts.Tile]]bool
	}
	var cases []found
	for s := uint64(1); s < 2000 && len(cases) < 20; s++ {
		for y := range consts.Chunk(30) {
			for x := range cols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if chunkTypeAt(raws, s, c, cols, 0) != chunkLandmark {
					continue
				}
				if r := chunkRoadTiles(raws, s, c, cols, 0, chunkW, chunkH); len(r) > 0 {
					cases = append(cases, found{s, c, r})
				}
			}
		}
	}
	require.GreaterOrEqual(t, len(cases), 20, "前提: 道の通るランドマークのチャンクが見つかる")

	placed := 0
	var blocked, covered []consts.Coord[consts.Tile]
	for _, tc := range cases {
		world := testutil.InitTestWorld(t)
		for y := range chunkH {
			for x := range chunkW {
				_, err := lifecycle.SpawnTile(world, "dirt_tundra", x, y, new(15))
				require.NoError(t, err)
			}
		}
		g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: chunkW, chunkH: chunkH, tiles: &tileIndex{world: world, loX: 0, hiX: chunkW}}
		require.NoError(t, wildernessLandmarkFeature{}.place(world, tc.seed, tc.c, cols, g))
		q := ecs.NewFilter2[gc.GridElement, gc.Prop](world.ECS).Query()
		for q.Next() {
			placed++
			if p := world.Components.GridElement.Get(q.Entity()).Coord; tc.roads[p] {
				blocked = append(blocked, p)
			}
		}
		// ランドマークは道より先に置くので、道のマスが土のままなら壁や床でも道を塞いでいない
		tiles := g.tiles.get()
		for p := range tc.roads {
			if id := tileIDAt(world, tiles, p); id != "dirt_tundra" {
				covered = append(covered, p)
			}
		}
	}
	require.Positive(t, placed, "前提: ランドマークが置かれる")
	assert.Empty(t, blocked, "道のマスにランドマークの置物が無い")
	assert.Empty(t, covered, "道のマスをランドマークの壁や床で塞がない")
}

func TestLandmarks_道が十字に通るチャンクでも全ての絵に置き場所がある(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	chunkW, chunkH, _, _ := dungeon.DungeonOverworld.BandShape()
	width := 0
	for _, f := range raw.PtrSlice(raws.PointFeatures) {
		if f.Road != nil {
			width = max(width, f.Road.Width)
		}
	}
	require.Positive(t, width, "前提: 道を持つ地物がある")
	roads := map[consts.Coord[consts.Tile]]bool{}
	cross := []roadSeg{
		span(orientHorizontal, chunkCenter(0, chunkH), 0, chunkW-1),
		span(orientVertical, chunkCenter(0, chunkW), 0, chunkH-1),
	}
	for p := range roadBand(cross, width) {
		roads[p] = true
	}
	for _, lm := range raw.PtrSlice(raws.Landmarks) {
		for _, name := range loader.FacilityChunkNames(lm.Id) {
			_, _, cells, err := loader.LoadTemplateByName(name, 1)
			require.NoError(t, err, name)
			// 絵は回転して置くので、幅と高さの大きい方で確かめる
			side := consts.Tile(max(len(cells), len(cells[0])))
			assert.NotEmpty(t, landmarkOrigins(side, side, chunkW, chunkH, roads), name)
		}
	}
}
