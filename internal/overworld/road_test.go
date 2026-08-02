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

	// 西集落の中心 Y の高さの水平路から、幅を持たせた横帯の1本を探す。列 x で west.Y を含む床の
	// 縦連続が丁度 roadWidth なら、他の道や垂直辺と重ならない単独の横帯である。地物が密になったため
	// 固定座標では市街地やPOIと重なりうるが、集落間の全区間が覆われることはない
	// road.go の非公開 roadWidth と一致させる。実装側を変えたらここも合わせる
	const roadWidth = 4
	west, east := centers[0], centers[1]
	isFloor := func(x, y consts.Tile) bool {
		return strings.HasPrefix(spriteKeyAtOrEmpty(world, x, y), consts.TileNameFloor)
	}
	found := false
	for x := west.X + 1; x < east.X; x++ {
		if !isFloor(x, west.Y) || !isFloor(x-1, west.Y) || !isFloor(x+1, west.Y) {
			continue // 水平に連続する床であること
		}
		// west.Y を含む縦の床連続を測る。交差などで厚みが違う列は単独の横帯でないので飛ばす
		top, bottom := west.Y, west.Y
		for isFloor(x, top-1) {
			top--
		}
		for isFloor(x, bottom+1) {
			bottom++
		}
		if bottom-top+1 != roadWidth {
			continue
		}
		// 幅 roadWidth の横帯。内部タイルは四方が床なのでオートタイル添字15になる。仮の 0 のままなら
		// 孤立タイル絵が並ぶ退行なので固定する。上下左右すべて床の内部タイルを選ぶ
		mid := top + 1
		if !isFloor(x-1, mid) || !isFloor(x+1, mid) || !isFloor(x, mid-1) || !isFloor(x, mid+1) {
			continue
		}
		key := spriteKeyAtOrEmpty(world, x, mid)
		assert.True(t, strings.HasSuffix(key, "_15"), "幅を持たせた道の内部は四方接続の添字15。実際: %q", key)
		found = true
		break
	}
	require.True(t, found, "集落間に幅 roadWidth の舗装区間がある")
}
