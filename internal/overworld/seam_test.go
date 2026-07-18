package overworld_test

import (
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChunkGen_継ぎ目は生成順に依存しない は、隣接2チャンクの境界オートタイルが
// 生成順（東シフト=西→東 / 西シフト=東→西）に依存しないことを固定する。
func TestChunkGen_継ぎ目は生成順に依存しない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	const runSeed uint64 = 42
	planner := mapplanner.PlannerTypeOverworldField

	// 西→東の順（通常の初期生成・東シフト相当）
	wEast := testutil.InitTestWorld(t)
	genA := overworld.NewChunkGen(wEast, runSeed, chunkW, chunkH, planner)
	require.NoError(t, genA(0, 0))
	require.NoError(t, genA(1, chunkW))

	// 東→西の順（西シフト相当: 東チャンクが既に在り、後から西端を生成）
	wWest := testutil.InitTestWorld(t)
	genB := overworld.NewChunkGen(wWest, runSeed, chunkW, chunkH, planner)
	require.NoError(t, genB(1, chunkW))
	require.NoError(t, genB(0, 0))

	// 境界2列(chunkW-1 = 西チャンク東端, chunkW = 東チャンク西端)の SpriteKey が一致する
	for _, x := range []consts.Tile{chunkW - 1, chunkW} {
		for y := range chunkH {
			ka := spriteKeyAtOrEmpty(wEast, x, y)
			kb := spriteKeyAtOrEmpty(wWest, x, y)
			assert.Equalf(t, ka, kb, "境界(%d,%d)の SpriteKey は生成順に依存しない", x, y)
		}
	}
}

// TestRecalcSeamAutotile_帯端は自己スキップ は、境界の片側にしかタイルが無い（帯の最端で
// 隣チャンクが無い）とき RecalcSeamAutotile が何もしないことを固定する。これにより呼び出し側は
// 東西どちらの境界かを気にせず両境界を無条件に呼べる。
func TestRecalcSeamAutotile_帯端は自己スキップ(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const boundaryX consts.Tile = 50

	// 境界の東側(boundaryX 以降)だけに dirt を敷く。西側(boundaryX-1)は空＝隣チャンク無し
	edge := 0
	for x := boundaryX; x <= boundaryX+1; x++ {
		for y := consts.Tile(4); y <= 6; y++ {
			_, err := lifecycle.SpawnTile(world, "dirt", x, y, &edge)
			require.NoError(t, err)
		}
	}
	before := spriteKeyAt(t, world, boundaryX, 5)

	overworld.RecalcSeamAutotile(world, boundaryX)

	after := spriteKeyAt(t, world, boundaryX, 5)
	assert.Equal(t, before, after, "片側が空の帯端では再計算せず SpriteKey を変えない")
}

// spriteKeyAtOrEmpty は指定座標のタイルの SpriteKey を返す。無ければ空文字。
func spriteKeyAtOrEmpty(world w.World, x, y consts.Tile) string {
	q := ecs.NewFilter2[gc.GridElement, gc.SpriteRender](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		g := world.Components.GridElement.Get(e)
		if g.X == x && g.Y == y {
			key := world.Components.SpriteRender.Get(e).SpriteKey
			q.Close()
			return key
		}
	}
	return ""
}

// TestRecalcSeamAutotile は境界2列のオートタイルが接合後に隣チャンクを見て再計算されることを固定する。
// 境界を跨いで dirt を敷き、端スプライト(_0)で生成した後に再計算すると、近傍がすべて dirt なので
// 全方向接続(_15)になる。
func TestRecalcSeamAutotile(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const boundaryX consts.Tile = 50

	// 境界の周囲に dirt の 4x3 ブロックを敷く（各境界タイルの4近傍が dirt になるように）。
	// 端スプライトを模して autoTileIndex=0 で生成する
	zero := 0
	for x := boundaryX - 2; x <= boundaryX+1; x++ {
		for y := consts.Tile(4); y <= 6; y++ {
			_, err := lifecycle.SpawnTile(world, "dirt", x, y, &zero)
			require.NoError(t, err)
		}
	}

	overworld.RecalcSeamAutotile(world, boundaryX)

	// 境界タイル (boundaryX-1, 5) と (boundaryX, 5) は4近傍すべて dirt なので _15 になる
	for _, bx := range []consts.Tile{boundaryX - 1, boundaryX} {
		key := spriteKeyAt(t, world, bx, 5)
		assert.Truef(t, strings.HasSuffix(key, "_15"),
			"境界タイル x=%d は近傍反映で全接続(_15)になる。実際: %s", bx, key)
	}
}

// spriteKeyAt は指定座標のタイルの SpriteKey を返す。
func spriteKeyAt(t *testing.T, world w.World, x, y consts.Tile) string {
	t.Helper()
	q := ecs.NewFilter2[gc.GridElement, gc.SpriteRender](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		g := world.Components.GridElement.Get(e)
		if g.X == x && g.Y == y {
			key := world.Components.SpriteRender.Get(e).SpriteKey
			q.Close()
			return key
		}
	}
	require.Failf(t, "タイルが見つからない", "(%d,%d)", x, y)
	return ""
}
