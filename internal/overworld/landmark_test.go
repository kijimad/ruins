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
