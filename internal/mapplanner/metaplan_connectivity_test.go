package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetaPlanConnectivityIntegration(t *testing.T) {
	t.Parallel()

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)
	world.Resources.RawMaster = createTownTestRawMaster()

	// 接続性検証が組み込まれたPlan関数をテスト
	width, height := 10, 10
	seed := uint64(42)
	plannerType := PlannerTypeSmallRoom

	// MetaPlanを生成（接続性検証込み）
	metaPlan, err := Plan(world, width, height, seed, plannerType)
	assert.NoError(t, err, "Plan with connectivity validation failed")
	assert.NotNil(t, metaPlan, "MetaPlan should not be nil")

	// プレイヤー開始位置が設定されていることを確認
	playerX, playerY, hasPlayer := metaPlan.GetPlayerStartPosition()
	assert.True(t, hasPlayer, "Should have player start position")
	assert.GreaterOrEqual(t, playerX, 0, "Player X should be valid")
	assert.GreaterOrEqual(t, playerY, 0, "Player Y should be valid")

	// 接続性を再検証（Plan関数内で既に検証済みだが、確認のため）
	pathFinder := NewPathFinder(metaPlan)
	err = pathFinder.ValidateConnectivity(playerX, playerY)
	assert.NoError(t, err, "Connectivity validation should pass")

	t.Logf("接続性検証統合テスト成功: プレイヤー位置=(%d,%d), ワープポータル数=%d",
		playerX, playerY, len(metaPlan.WarpPortals))
}

func TestMetaPlanConnectivityWithTownMap(t *testing.T) {
	t.Parallel()

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)
	world.Resources.RawMaster = createTownTestRawMaster()

	// 街マップでの接続性検証テスト
	width, height := 50, 50
	seed := uint64(123)
	plannerType := PlannerTypeTown

	// MetaPlanを生成（接続性検証込み）
	metaPlan, err := Plan(world, width, height, seed, plannerType)
	assert.NoError(t, err, "Town plan with connectivity validation failed")
	assert.NotNil(t, metaPlan, "Town MetaPlan should not be nil")

	// プレイヤー開始位置の確認
	playerX, playerY, hasPlayer := metaPlan.GetPlayerStartPosition()
	assert.True(t, hasPlayer, "Town should have player start position")

	// 接続性検証
	pathFinder := NewPathFinder(metaPlan)
	err = pathFinder.ValidateConnectivity(playerX, playerY)
	assert.NoError(t, err, "Town connectivity validation should pass")

	t.Logf("街マップ接続性検証成功: プレイヤー位置=(%d,%d), Props数=%d",
		playerX, playerY, len(metaPlan.Props))
}
