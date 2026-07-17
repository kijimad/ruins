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

// columnHasPassable は列 x に通行可能タイルが1つ以上あるかを返す。
func columnHasPassable(plan *mapplanner.MetaPlan, x, h int) bool {
	for y := range h {
		idx := plan.Level.XYTileIndex(consts.Tile(x), consts.Tile(y))
		if !plan.Tiles[idx].BlockPass {
			return true
		}
	}
	return false
}

func TestOverworldField_全列が通行可能(t *testing.T) {
	t.Parallel()

	const w, h consts.Tile = 80, 40
	// 複数 seed で「どの列も高さ全体を塞がない」通行保証を確認する
	for _, seed := range []uint64{1, 7, 42, 100, 999} {
		plan := planField(t, w, h, seed)
		for x := range int(w) {
			assert.Truef(t, columnHasPassable(plan, x, int(h)),
				"seed=%d: 列 x=%d に通行可能タイルが無い（東西通行が塞がれる）", seed, x)
		}
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
