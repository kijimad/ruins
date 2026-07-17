package mapplanner_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// planField は開けた地形チャンクを生成して MetaPlan を返す。
func planField(t *testing.T, w, h consts.Tile, seed uint64) *mapplanner.MetaPlan {
	t.Helper()
	world := testutil.InitTestWorld(t)
	plan, err := mapplanner.Plan(world, w, h, seed, mapplanner.PlannerTypeOverworldField)
	require.NoError(t, err)
	return plan
}

// eastWestConnected は西端列(x=0)の通行可能タイルから4連結で東端列(x=w-1)に到達できるかを返す。
// per-column passability ではなく「実際に西端→東端を歩けるか」を BFS で検証する。
func eastWestConnected(plan *mapplanner.MetaPlan, w, h int) bool {
	passable := func(x, y int) bool {
		if x < 0 || x >= w || y < 0 || y >= h {
			return false
		}
		return !plan.Tiles[plan.Level.XYTileIndex(consts.Tile(x), consts.Tile(y))].BlockPass
	}
	visited := make([]bool, w*h)
	var queue [][2]int
	for y := range h {
		if passable(0, y) {
			queue = append(queue, [2]int{0, y})
			visited[y*w] = true
		}
	}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		x, y := c[0], c[1]
		if x == w-1 {
			return true // 東端に到達
		}
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if passable(nx, ny) && !visited[ny*w+nx] {
				visited[ny*w+nx] = true
				queue = append(queue, [2]int{nx, ny})
			}
		}
	}
	return false
}

func TestOverworldField_東西が連結する(t *testing.T) {
	t.Parallel()

	const w, h consts.Tile = 80, 40
	// 複数 seed で「西端から東端まで実際に4連結で歩ける」ことを検証する（連結性の構造保証）
	for _, seed := range []uint64{1, 7, 42, 100, 999, 12345, 67890} {
		plan := planField(t, w, h, seed)
		assert.Truef(t, eastWestConnected(plan, int(w), int(h)),
			"seed=%d: 西端から東端へ4連結の経路が無い（東西通行が塞がれる）", seed)
	}
}

func TestOverworldField_通行可能がデフォルト(t *testing.T) {
	t.Parallel()

	const w, h consts.Tile = 80, 40
	plan := planField(t, w, h, 42)

	passable := 0
	for _, tile := range plan.Tiles {
		if !tile.BlockPass {
			passable++
		}
	}
	// 開けた地形なので大半が通行可能（障壁は例外）
	ratio := float64(passable) / float64(len(plan.Tiles))
	assert.Greater(t, ratio, 0.7, "通行可能タイルが大半を占める（開けた地形）")
}

func TestOverworldField_決定的(t *testing.T) {
	t.Parallel()

	const w, h consts.Tile = 60, 30
	a := planField(t, w, h, 555)
	b := planField(t, w, h, 555)
	require.Len(t, b.Tiles, len(a.Tiles))
	for i := range a.Tiles {
		require.Equalf(t, a.Tiles[i].Name, b.Tiles[i].Name, "同一 seed は同一レイアウト (idx=%d)", i)
	}
}
