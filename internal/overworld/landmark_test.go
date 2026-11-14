package overworld

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrawers_全DrawerKeyに実装がある(t *testing.T) {
	t.Parallel()

	for _, key := range rawschema.DrawerKeyValues() {
		_, ok := drawers[key]
		assert.Truef(t, ok, "drawer key %q に Go 実装がある", key)
	}
}

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

func TestDrawHut_小屋の床は屋内になる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	g := chunkGeom{chunkW: 8, chunkH: 8, tiles: &tileIndex{world: world, loX: 0, hiX: 8}}
	require.NoError(t, drawHut(world, g, rand.New(rand.NewPCG(1, 0)), consts.Coord[consts.Tile]{X: 1, Y: 1}, rawschema.LandmarkHut{W: 4, H: 4, Floor: consts.TileNameFloor, Wall: consts.TileNameDWall}, nil))

	got := map[string]map[rawschema.Shelter]int{}
	for _, e := range g.tiles.get() {
		id := world.Components.RawID.Get(e).ID
		if got[id] == nil {
			got[id] = map[rawschema.Shelter]int{}
		}
		got[id][world.Components.TileEnvironment.Get(e).Shelter]++
	}
	assert.Equal(t, map[string]map[rawschema.Shelter]int{
		consts.TileNameFloor: {rawschema.Indoor: 5},
		consts.TileNameDWall: {rawschema.Outdoor: 11},
	}, got)
}

func TestDrawHutLandmark_小屋の形が無いとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	g := chunkGeom{chunkW: 8, chunkH: 8, tiles: &tileIndex{world: world, loX: 0, hiX: 8}}
	lm := rawschema.Landmark{Id: "hut", Drawer: rawschema.Hut}
	err := drawHutLandmark(world, g, rand.New(rand.NewPCG(1, 0)), consts.Coord[consts.Tile]{X: 1, Y: 1}, lm)
	require.ErrorIs(t, err, errHutMissing)
}
