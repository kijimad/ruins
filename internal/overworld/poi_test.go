package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findPOIChunk はPOIが当選し、他の地物に譲らないチャンクと seed を探す。
func findPOIChunk(t *testing.T) (uint64, consts.Coord[consts.Chunk]) {
	t.Helper()
	const rows = 1
	for s := uint64(1); s < 200; s++ {
		for x := range consts.Chunk(30) {
			c := consts.Coord[consts.Chunk]{X: x}
			if !poiPlacement.At(s, c, rows) || settlementPlacement.At(s, c, rows) || ruinPlacement.At(s, c, rows) {
				continue
			}
			if _, _, _, ok := urbanRegionOf(s, c, rows); ok {
				continue
			}
			return s, c
		}
	}
	require.Fail(t, "前提: POIだけが当選するチャンクが見つかる")
	return 0, consts.Coord[consts.Chunk]{}
}

// poiEntities は名前を持つエンティティを 座標 → 名前 の対応で集める。座標を文字列へ畳まず
// consts.Coord をそのまま map のキーにするので、桁や区切りの表記に依存せず整列も要らない。
func poiEntities(world w.World) map[consts.Coord[consts.Tile]]string {
	got := map[consts.Coord[consts.Tile]]string{}
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		got[world.Components.GridElement.Get(e).Coord] = world.Components.Name.Get(e).Name
	}
	return got
}

func TestWildernessPOI_原野の当選チャンクに小構造物が決定的に出る(t *testing.T) {
	t.Parallel()

	seed, c := findPOIChunk(t)

	build := func() map[consts.Coord[consts.Tile]]string {
		world := testutil.InitTestWorld(t)
		g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: 50, chunkH: 50, tiles: &tileIndex{world: world, loX: 0, hiX: 50}}
		require.NoError(t, wildernessPOIFeature{}.place(world, seed, c, 1, g))
		return poiEntities(world)
	}
	a := build()
	b := build()
	assert.NotEmpty(t, a, "POIの prop が置かれる")
	assert.Equal(t, a, b, "POIの配置は決定的で再生成しても一致する")
}
