package overworld_test

import (
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
