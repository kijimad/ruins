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

// TestNewChunkGen_隣接する小集落が道で結ばれる は、隣接リージョンの集落中心どうしを
// 結ぶ舗装路が生成されることを固定する。集落位置は WinnerOf の純関数算出に依るため、
// 中間チャンクは両端の集落を生成せずとも道を描ける。
func TestNewChunkGen_隣接する小集落が道で結ばれる(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	world := testutil.InitTestWorld(t)
	const runSeed = 27
	gen := overworld.NewChunkGen(world, runSeed, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	// 集落は Y 方向のリージョンに並ぶので Y 方向に生成する
	for i := range 16 {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{Y: consts.Chunk(i)}, 0, consts.Tile(i)*chunkH))
	}

	// 集落に分類されるチャンクの中心を集める。道は集落のチャンクの中心どうしを結ぶ
	var centers []consts.Coord[consts.Tile]
	for i := range 16 {
		if overworld.IsSettlementChunk(world.Resources.RawMaster, runSeed, consts.Coord[consts.Chunk]{Y: consts.Chunk(i)}, 1) {
			centers = append(centers, consts.Coord[consts.Tile]{X: chunkW / 2, Y: consts.Tile(i)*chunkH + chunkH/2})
		}
	}
	require.GreaterOrEqual(t, len(centers), 2, "前提: 隣接リージョンに集落が2つある")

	// 集落は Y 隣接なので道は縦に走る。集落中心 X の列で、幅を持たせた縦帯の1本を探す。行 y で
	// north.X を含む床の横連続が丁度 roadWidth なら、他の道や水平辺と重ならない単独の縦帯である。
	var roadWidth consts.Tile
	for _, f := range raw.PtrSlice(world.Resources.RawMaster.PointFeatures) {
		if f.Id == rawschema.PointFeatureKindSettlement {
			roadWidth = consts.Tile(f.Road.Width)
		}
	}
	north, south := centers[0], centers[1]
	isFloor := func(x, y consts.Tile) bool {
		return strings.HasPrefix(spriteKeyAtOrEmpty(world, x, y), consts.TileNameFloor)
	}
	found := false
	for y := north.Y + 1; y < south.Y; y++ {
		if !isFloor(north.X, y) || !isFloor(north.X, y-1) || !isFloor(north.X, y+1) {
			continue // 垂直に連続する床であること
		}
		// north.X を含む横の床連続を測る。交差などで厚みが違う行は単独の縦帯でないので飛ばす
		left, right := north.X, north.X
		for isFloor(left-1, y) {
			left--
		}
		for isFloor(right+1, y) {
			right++
		}
		if right-left+1 != roadWidth {
			continue
		}
		// 幅 roadWidth の縦帯。内部タイルは四方が床なのでオートタイル添字15になる。仮の 0 のままなら
		// 孤立タイル絵が並ぶ退行なので固定する。上下左右すべて床の内部タイルを選ぶ
		mid := left + 1
		if !isFloor(mid-1, y) || !isFloor(mid+1, y) || !isFloor(mid, y-1) || !isFloor(mid, y+1) {
			continue
		}
		key := spriteKeyAtOrEmpty(world, mid, y)
		assert.True(t, strings.HasSuffix(key, "_15"), "幅を持たせた道の内部は四方接続の添字15。実際: %q", key)
		found = true
		break
	}
	require.True(t, found, "集落間に幅 roadWidth の舗装区間がある")
}

func TestNewChunkGen_集落のチャンクの中は道を舗装しない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	world := testutil.InitTestWorld(t)
	const runSeed = 27
	gen := overworld.NewChunkGen(world, runSeed, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	for i := range 16 {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{Y: consts.Chunk(i)}, 0, consts.Tile(i)*chunkH))
	}

	// 1列の世界では道は集落の中心の列を縦に走り、集落のチャンクの北と南の端の行を通る。端の行は施設の絵の外にある
	var edges []string
	var want []string
	for i := range 16 {
		if !overworld.IsSettlementChunk(world.Resources.RawMaster, runSeed, consts.Coord[consts.Chunk]{Y: consts.Chunk(i)}, 1) {
			continue
		}
		for _, y := range []consts.Tile{consts.Tile(i) * chunkH, consts.Tile(i+1)*chunkH - 1} {
			key := spriteKeyAtOrEmpty(world, chunkW/2, y)
			edges = append(edges, strings.SplitN(key, "_", 2)[0])
			want = append(want, consts.TileNameDirt)
		}
	}
	require.GreaterOrEqual(t, len(edges), 4, "前提: 集落が2つ以上ある")
	assert.Equal(t, want, edges)
}
