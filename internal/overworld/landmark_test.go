package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
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
		for x := range consts.Chunk(30) {
			c := consts.Coord[consts.Chunk]{X: x}
			if !landmarkPlacement.At(s, c, rows) || settlementPlacement.At(s, c, rows) || dungeonEntrancePlacement.At(s, c, rows) {
				continue
			}
			if _, ok := districtAt(raws, s, c, rows); ok {
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
	_, _, err := spawnFacility(world, g, cells, 3, 3, "shrine", 1, 1)
	require.NoError(t, err, "右下の端までちょうど収まる")
	_, _, err = spawnFacility(world, g, cells, 4, 3, "shrine", 1, 1)
	require.ErrorIs(t, err, errFacilityOutOfChunk)
}
