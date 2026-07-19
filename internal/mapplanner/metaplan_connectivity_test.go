package mapplanner

import (
	"fmt"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectivity_AllPlannerTypes は全プランナータイプで接続性を検証する。
// 多数のシードで生成し、全ポータルに到達可能なことを確認する
func TestConnectivity_AllPlannerTypes(t *testing.T) {
	t.Parallel()

	// プロシージャルマップのプランナータイプ（ポータルを動的配置するもの）
	proceduralPlanners := []struct {
		name        string
		plannerType PlannerType
	}{
		{"小部屋", PlannerTypeSmallRoom},
		{"大部屋", PlannerTypeBigRoom},
		{"洞窟", PlannerTypeCave},
		{"廃墟", PlannerTypeRuins},
		{"森", PlannerTypeForest},
	}

	seedCount := 30

	for _, tc := range proceduralPlanners {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for seed := range uint64(seedCount) {
				t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
					t.Parallel()
					world := testutil.InitTestWorld(t)
					world.Resources.RawMaster = *CreateTestRawMaster()
					query.SetDungeon(world, &gc.Dungeon{Depth: 5}) // EscapePortalも生成される階層

					plan, err := Plan(world, 50, 50, seed, tc.plannerType)
					require.NoError(t, err, "Plan失敗")

					assertMapConnectivity(t, plan)
				})
			}
		})
	}
}

// TestConnectivity_TemplatePlanners はテンプレートベースのプランナーで接続性を検証する
func TestConnectivity_TemplatePlanners(t *testing.T) {
	t.Parallel()

	templatePlanners := []struct {
		name        string
		plannerType PlannerType
	}{
		{"ボスフロア", PlannerTypeBossFloor},
		{"市街地", PlannerTypeTown},
		{"広場", PlannerTypeTownPlaza},
	}

	for _, tc := range templatePlanners {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			world.Resources.RawMaster = *CreateTestRawMaster()
			query.SetDungeon(world, &gc.Dungeon{Depth: 5})

			plan, err := Plan(world, 50, 50, 12345, tc.plannerType)
			require.NoError(t, err, "Plan失敗")

			// プレイヤー開始位置が歩行可能であることを確認
			playerPos, err := plan.GetPlayerStartPosition()
			require.NoError(t, err, "プレイヤー開始位置が未設定")

			pf := NewPathFinder(plan)
			assert.True(t, pf.IsWalkable(playerPos),
				"プレイヤー開始位置(%d,%d)が歩行不可", playerPos.X, playerPos.Y)

			// テンプレートのポータルはProps経由で配置されるので、MetaPlan上のポータルリストではなく
			// プレイヤーから到達可能な歩行可能タイルが十分にあることを確認する
			reachableCount := countReachableTiles(pf, playerPos)
			assert.Greater(t, reachableCount, 10,
				"プレイヤーから到達可能なタイルが少なすぎる（%d）", reachableCount)
		})
	}
}

// assertMapConnectivity はMetaPlanの接続性を包括的に検証する
func assertMapConnectivity(t *testing.T, plan *MetaPlan) {
	t.Helper()

	playerPos, err := plan.GetPlayerStartPosition()
	require.NoError(t, err, "プレイヤー開始位置が未設定")

	pf := NewPathFinder(plan)

	// プレイヤー開始位置が歩行可能であること
	assert.True(t, pf.IsWalkable(playerPos),
		"プレイヤー開始位置(%d,%d)が歩行不可", playerPos.X, playerPos.Y)

	// NextPortalが存在し、到達可能であること
	require.NotEmpty(t, plan.NextPortals, "NextPortalが存在しない")
	for i, portal := range plan.NextPortals {
		assert.True(t, pf.IsWalkable(portal),
			"NextPortal[%d](%d,%d)が歩行不可タイル上にある", i, portal.X, portal.Y)
		assert.True(t, pf.IsReachable(playerPos, portal),
			"NextPortal[%d](%d,%d)にプレイヤー(%d,%d)から到達不可", i, portal.X, portal.Y, playerPos.X, playerPos.Y)
	}

	// EscapePortalが存在する場合、到達可能であること
	for i, portal := range plan.EscapePortals {
		assert.True(t, pf.IsWalkable(portal),
			"EscapePortal[%d](%d,%d)が歩行不可タイル上にある", i, portal.X, portal.Y)
		assert.True(t, pf.IsReachable(playerPos, portal),
			"EscapePortal[%d](%d,%d)にプレイヤー(%d,%d)から到達不可", i, portal.X, portal.Y, playerPos.X, playerPos.Y)
	}
}

// countReachableTiles はBFSで指定位置から到達可能な歩行可能タイル数を返す
func countReachableTiles(pf *PathFinder, start consts.Coord[consts.Tile]) int {
	return pf.countReachableFrom(start)
}
