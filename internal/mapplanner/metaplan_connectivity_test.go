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
	world.Resources.RawMaster = CreateTestRawMaster()

	// 接続性検証が組み込まれたPlan関数をテスト
	width, height := 50, 50
	seed := uint64(42)
	plannerType := PlannerTypeSmallRoom

	// MetaPlanを生成（接続性検証込み）
	metaPlan, err := Plan(world, width, height, &seed, plannerType)
	assert.NoError(t, err, "Plan with connectivity validation failed")
	assert.NotNil(t, metaPlan, "MetaPlan should not be nil")

	// プレイヤー開始位置が設定されていることを確認
	playerPos, err := metaPlan.GetPlayerStartPosition()
	assert.NoError(t, err, "Should have player start position")
	assert.GreaterOrEqual(t, playerPos.X, 0, "Player X should be valid")
	assert.GreaterOrEqual(t, playerPos.Y, 0, "Player Y should be valid")

	t.Logf("接続性検証統合テスト成功: プレイヤー位置=(%d,%d)",
		playerPos.X, playerPos.Y)
}
