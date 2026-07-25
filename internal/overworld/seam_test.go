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
	genA := overworld.NewChunkGen(wEast, runSeed, chunkW, chunkH, 1, planner)
	require.NoError(t, genA(consts.Coord[consts.Chunk]{X: 0}, 0, 0))
	require.NoError(t, genA(consts.Coord[consts.Chunk]{X: 1}, chunkW, 0))

	// 東→西の順（西シフト相当: 東チャンクが既に在り、後から西端を生成）
	wWest := testutil.InitTestWorld(t)
	genB := overworld.NewChunkGen(wWest, runSeed, chunkW, chunkH, 1, planner)
	require.NoError(t, genB(consts.Coord[consts.Chunk]{X: 1}, chunkW, 0))
	require.NoError(t, genB(consts.Coord[consts.Chunk]{X: 0}, 0, 0))

	// 境界2列(chunkW-1 = 西チャンク東端, chunkW = 東チャンク西端)の SpriteKey が一致する
	for _, x := range []consts.Tile{chunkW - 1, chunkW} {
		for y := range chunkH {
			ka := spriteKeyAtOrEmpty(wEast, x, y)
			kb := spriteKeyAtOrEmpty(wWest, x, y)
			assert.Equalf(t, ka, kb, "境界(%d,%d)の SpriteKey は生成順に依存しない", x, y)
		}
	}
}

// TestRecalcChunkSeams_帯端は自己スキップ は、境界の片側にしかタイルが無い（帯の最端で
// 隣チャンクが無い）とき、その境界を再計算せず何もしないことを固定する。これにより呼び出し側は
// 東西南北どの境界かを気にせず4境界を無条件に呼べる。
func TestRecalcChunkSeams_帯端は自己スキップ(t *testing.T) {
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

	// 原点を境界に置き寸法を大きく取ると、検証したい x=boundaryX 以外の3境界はタイルが無く自己スキップする
	overworld.RecalcChunkSeams(world, boundaryX, 0, 100, 100)

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

// TestRecalcChunkSeams_東西境界は隣を見て再計算する は、境界2列のオートタイルが接合後に
// 隣チャンクを見て再計算されることを固定する。境界を跨いで dirt を敷き、端スプライト(_0)で
// 生成した後に再計算すると、近傍がすべて dirt なので全方向接続(_15)になる。
func TestRecalcChunkSeams_東西境界は隣を見て再計算する(t *testing.T) {
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

	// 原点を x=boundaryX に置き寸法を大きく取ると、他3境界はタイルが無く自己スキップし東西境界だけが対象になる
	overworld.RecalcChunkSeams(world, boundaryX, 0, 100, 100)

	// 境界タイル (boundaryX-1, 5) と (boundaryX, 5) は4近傍すべて dirt なので _15 になる
	for _, bx := range []consts.Tile{boundaryX - 1, boundaryX} {
		key := spriteKeyAt(t, world, bx, 5)
		assert.Truef(t, strings.HasSuffix(key, "_15"),
			"境界タイル x=%d は近傍反映で全接続(_15)になる。実際: %s", bx, key)
	}
}

// TestRecalcChunkSeams_南北境界は隣を見て再計算する は、南北境界2行のオートタイルが接合後に
// 隣チャンクを見て再計算されることを固定する。
func TestRecalcChunkSeams_南北境界は隣を見て再計算する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const boundaryY consts.Tile = 40

	// 境界の周囲に dirt の 3x4 ブロックを敷く。端スプライトを模して autoTileIndex=0 で生成する
	zero := 0
	for y := boundaryY - 2; y <= boundaryY+1; y++ {
		for x := consts.Tile(4); x <= 6; x++ {
			_, err := lifecycle.SpawnTile(world, "dirt", x, y, &zero)
			require.NoError(t, err)
		}
	}

	// 原点を y=boundaryY に置き寸法を大きく取ると、他3境界はタイルが無く自己スキップし南北境界だけが対象になる
	overworld.RecalcChunkSeams(world, 0, boundaryY, 100, 100)

	// 境界タイル (5, boundaryY-1) と (5, boundaryY) は4近傍すべて dirt なので _15 になる
	for _, by := range []consts.Tile{boundaryY - 1, boundaryY} {
		key := spriteKeyAt(t, world, 5, by)
		assert.Truef(t, strings.HasSuffix(key, "_15"),
			"境界タイル y=%d は近傍反映で全接続(_15)になる。実際: %s", by, key)
	}
}

// TestChunkGen_縦の継ぎ目は生成順に依存しない は、縦に積んだ2チャンクの境界2行が
// 生成順（上→下 / 下→上）に依存しないことを固定する。行が増えたときの縦版の対称性。
func TestChunkGen_縦の継ぎ目は生成順に依存しない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	const runSeed uint64 = 42
	planner := mapplanner.PlannerTypeOverworldField

	// 上→下の順
	wTop := testutil.InitTestWorld(t)
	genA := overworld.NewChunkGen(wTop, runSeed, chunkW, chunkH, 1, planner)
	require.NoError(t, genA(consts.Coord[consts.Chunk]{X: 0, Y: 0}, 0, 0))
	require.NoError(t, genA(consts.Coord[consts.Chunk]{X: 0, Y: 1}, 0, chunkH))

	// 下→上の順
	wBottom := testutil.InitTestWorld(t)
	genB := overworld.NewChunkGen(wBottom, runSeed, chunkW, chunkH, 1, planner)
	require.NoError(t, genB(consts.Coord[consts.Chunk]{X: 0, Y: 1}, 0, chunkH))
	require.NoError(t, genB(consts.Coord[consts.Chunk]{X: 0, Y: 0}, 0, 0))

	// 境界2行(chunkH-1 = 上チャンク南端, chunkH = 下チャンク北端)の SpriteKey が一致する
	for _, y := range []consts.Tile{chunkH - 1, chunkH} {
		for x := range chunkW {
			ka := spriteKeyAtOrEmpty(wTop, x, y)
			kb := spriteKeyAtOrEmpty(wBottom, x, y)
			assert.Equalf(t, ka, kb, "境界(%d,%d)の SpriteKey は生成順に依存しない", x, y)
		}
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
