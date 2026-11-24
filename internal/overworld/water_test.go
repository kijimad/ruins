package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaterFeature_ノイズが水を示す土の地面だけを水面にする(t *testing.T) {
	t.Parallel()

	const seed uint64 = 1
	const chunkW, chunkH consts.Tile = 24, 24
	world := testutil.InitTestWorld(t)
	raws := world.Resources.RawMaster

	// ノイズが水を示すマスを持つ、荒れ地か道のチャンクを探す
	var c consts.Coord[consts.Chunk]
	found := false
	for y := consts.Chunk(0); y < 60 && !found; y++ {
		c = consts.Coord[consts.Chunk]{Y: y}
		switch chunkTypeAt(raws, seed, c, 1, 0) {
		case chunkWasteland, chunkRoad:
		case chunkCrashSite, chunkUrban, chunkSettlement, chunkLandmark:
			continue
		}
		for i := range chunkW * chunkH {
			if waterTileAt(seed^waterSalt, consts.Coord[consts.Tile]{X: i % chunkW, Y: c.Y.Tiles(chunkH) + i/chunkW}) != "" {
				found = true
				break
			}
		}
	}
	require.True(t, found, "前提: 水を持つ荒れ地か道のチャンクがある")

	gen := NewChunkGen(world, seed, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	require.NoError(t, gen(c, 0, 0))

	earth := earthTilesOf(raws)
	waters := 0
	q := query.ActiveFilter3[gc.GridElement, gc.Tile, gc.RawID](world).Query()
	for q.Next() {
		e := q.Entity()
		g := world.Components.GridElement.Get(e).Coord
		id := world.Components.RawID.Get(e).ID
		want := waterTileAt(seed^waterSalt, consts.Coord[consts.Tile]{X: g.X, Y: c.Y.Tiles(chunkH) + g.Y})
		switch {
		case id == waterShallowTile || id == waterDeepTile:
			// 水面はノイズが示すとおりの深さで置く
			assert.Equal(t, want, id, "%v", g)
			waters++
		case want != "":
			// ノイズが水を示しても水にならないのは、道の舗装など土でない地面だけ
			assert.False(t, earth[id], "%v は土の地面 %s のまま", g, id)
		}
	}
	assert.Positive(t, waters)
}
