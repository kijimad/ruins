package overworld_test

import (
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChunkGen_集落どうしを結ぶ道はチャンクの境目を通り施設の脇を過ぎる(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	const cols consts.Chunk = 2
	world := testutil.InitTestWorld(t)
	const runSeed = 2
	gen := overworld.NewChunkGen(world, runSeed, chunkW, chunkH, cols, mapplanner.PlannerTypeOverworldField)
	for y := range 8 {
		for x := range cols {
			require.NoError(t, gen(consts.Coord[consts.Chunk]{X: x, Y: consts.Chunk(y)}, x.Tiles(chunkW), consts.Tile(y)*chunkH))
		}
	}
	north, south := consts.Coord[consts.Chunk]{X: 0, Y: 0}, consts.Coord[consts.Chunk]{X: 1, Y: 6}
	for _, c := range []consts.Coord[consts.Chunk]{north, south} {
		require.True(t, overworld.IsSettlementChunk(world.Resources.RawMaster, runSeed, c, cols), "前提: %v は集落", c)
	}

	// 道は北の集落から南の集落の列の西の境目 x=24 を縦に通る。北の集落の施設はチャンクの中に収まるので、
	// 道は施設の東の脇を北の集落のチャンクの端から端まで過ぎる
	var roadWidth int
	for _, f := range raw.PtrSlice(world.Resources.RawMaster.PointFeatures) {
		if f.Id == rawschema.PointFeatureKindSettlement {
			roadWidth = f.Road.Width
		}
	}
	lane := south.X.Tiles(chunkW)
	var paved []string
	var want []string
	for y := north.Y.Tiles(chunkH); y < south.Y.Tiles(chunkH); y++ {
		for x := lane - consts.Tile(roadWidth)/2; x < lane+consts.Tile(roadWidth)/2; x++ {
			paved = append(paved, strings.SplitN(spriteKeyAtOrEmpty(world, x, y), "_", 2)[0])
			want = append(want, consts.TileNameFloor)
		}
	}
	assert.Equal(t, want, paved)

	// 幅を持たせた道の内部は四方が道なので、オートタイルの添字は15になる
	assert.Equal(t, consts.TileNameFloor+"_15", spriteKeyAtOrEmpty(world, lane, 3*chunkH))
}
