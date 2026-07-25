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
	gen := overworld.NewChunkGen(world, 321, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	for i := range 16 {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
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

	// 西集落の中心 Y の高さの水平路から、上下が原野のままの区間を探す。地物が密になった
	// ため固定座標では市街地やPOIと重なりうるが、集落間の全区間が覆われることはない
	west, east := centers[0], centers[1]
	isFloor := func(x, y consts.Tile) bool {
		return strings.HasPrefix(spriteKeyAtOrEmpty(world, x, y), consts.TileNameFloor)
	}
	found := false
	for x := west.X + 1; x < east.X; x++ {
		if !isFloor(x, west.Y) || !isFloor(x-1, west.Y) || !isFloor(x+1, west.Y) ||
			isFloor(x, west.Y-1) || isFloor(x, west.Y+1) {
			continue
		}
		// 左右にだけ床が続く区間なので、オートタイルは左8|右2=10 になる。
		// 添字が仮の 0 のままなら孤立タイル絵が並ぶ退行なので、ここで固定する
		key := spriteKeyAtOrEmpty(world, x, west.Y)
		assert.True(t, strings.HasSuffix(key, "_10"), "水平路の中間は左右接続の添字10。実際: %q", key)
		found = true
		break
	}
	require.True(t, found, "集落間に上下が原野のままの舗装区間がある")
}
