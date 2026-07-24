package overworld_test

import (
	"sort"
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewChunkGen_隣接する小集落が道で結ばれる は、隣接リージョンの集落中心どうしを
// 結ぶ舗装路が生成されることを固定する。集落位置は WinnerOf の純関数算出に依るため、
// 中間チャンクは両端の集落を生成せずとも道を描ける。
func TestNewChunkGen_隣接する小集落が道で結ばれる(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, 321, chunkW, chunkH, 1, worldstream.ChunkCoord{X: -1}, mapplanner.PlannerTypeOverworldField)
	for i := range 16 {
		require.NoError(t, gen(worldstream.ChunkCoord{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}

	// 商人の位置から集落中心を復元する。商人は中心の (-2,-1) に立つ
	var centers []consts.Coord[consts.Tile]
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.Name.Get(e).Name != merchantName {
			continue
		}
		p := world.Components.GridElement.Get(e).Coord
		centers = append(centers, consts.Coord[consts.Tile]{X: p.X + 2, Y: p.Y + 1})
	}
	sort.Slice(centers, func(i, j int) bool { return centers[i].X < centers[j].X })
	require.GreaterOrEqual(t, len(centers), 2, "前提: 隣接リージョンに集落が2つある")

	// 西集落の中心 Y の高さで、両集落の中間地点が舗装されている
	west, east := centers[0], centers[1]
	mid := (west.X + east.X) / 2
	key := spriteKeyAtOrEmpty(world, mid, west.Y)
	assert.True(t, strings.HasPrefix(key, consts.TileNameFloor),
		"集落間の中間 (%d,%d) が舗装される。実際: %q", mid, west.Y, key)
}
