package overworld_test

import (
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChunkGen_道は集落の脇の道ばたのチャンクを通る(t *testing.T) {
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
	settlement, roadside := consts.Coord[consts.Chunk]{X: 1, Y: 6}, consts.Coord[consts.Chunk]{X: 0, Y: 6}
	require.True(t, overworld.IsSettlementChunk(world.Resources.RawMaster, runSeed, settlement, cols), "前提: %v は集落", settlement)

	// 北の集落から来る道は、南の集落の西隣のチャンクへ北の縁から入り、チャンクの中心まで縦に通る
	cx := roadside.X.Tiles(chunkW) + chunkW/2
	var paved, want []string
	for y := roadside.Y.Tiles(chunkH); y <= roadside.Y.Tiles(chunkH)+chunkH/2; y++ {
		paved = append(paved, strings.SplitN(spriteKeyAtOrEmpty(world, cx, y), "_", 2)[0])
		want = append(want, consts.TileNameFloor)
	}
	assert.Equal(t, want, paved)
	// 幅を持たせた道の内部は四方が道なので、オートタイルの添字は15になる
	assert.Equal(t, consts.TileNameFloor+"_15", spriteKeyAtOrEmpty(world, cx, roadside.Y.Tiles(chunkH)+3))
}
