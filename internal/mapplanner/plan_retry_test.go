package mapplanner

import (
	"fmt"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// TestPlan_RetryLogic はPlan関数の再試行ロジックをテストする
func TestPlan_RetryLogic(t *testing.T) {
	t.Parallel()

	t.Run("正常なマップ生成では再試行なし", func(t *testing.T) {
		t.Parallel()
		w := testutil.InitTestWorld(t)
		w.Resources.RawMaster = CreateTestRawMaster()

		// 正常に生成できるプランナーでテスト
		plan, err := Plan(w, 20, 20, 12345, PlannerTypeSmallRoom)
		assert.NoError(t, err)
		assert.NotNil(t, plan)

		// プレイヤー位置が設定されていることを確認
		playerX, playerY, hasPlayer := plan.GetPlayerStartPosition()
		assert.True(t, hasPlayer)
		assert.GreaterOrEqual(t, playerX, 0)
		assert.GreaterOrEqual(t, playerY, 0)
	})

	t.Run("BigRoomPlannerでも正常動作", func(t *testing.T) {
		t.Parallel()
		w := testutil.InitTestWorld(t)
		w.Resources.RawMaster = CreateTestRawMaster()

		plan, err := Plan(w, 30, 30, 54321, PlannerTypeBigRoom)
		assert.NoError(t, err)
		assert.NotNil(t, plan)
	})

	t.Run("CavePlannerでも正常動作", func(t *testing.T) {
		t.Parallel()
		w := testutil.InitTestWorld(t)
		w.Resources.RawMaster = CreateTestRawMaster()

		plan, err := Plan(w, 25, 25, 99999, PlannerTypeCave)
		assert.NoError(t, err)
		assert.NotNil(t, plan)
	})

	t.Run("接続性検証の定数値確認", func(t *testing.T) {
		t.Parallel()
		// 再試行回数が適切に設定されていることを確認
		assert.Equal(t, 10, MaxPlanRetries)
	})
}

// TestPlan_ConnectivityValidation は接続性検証が動作することをテストする
func TestPlan_ConnectivityValidation(t *testing.T) {
	t.Parallel()

	t.Run("接続性検証が実行されることを確認", func(t *testing.T) {
		t.Parallel()
		w := testutil.InitTestWorld(t)
		w.Resources.RawMaster = CreateTestRawMaster()

		// 複数の異なるシードでテストして、すべて接続性チェックをパス
		seeds := []uint64{1, 100, 1000, 10000, 50000}
		for _, seed := range seeds {
			plan, err := Plan(w, 15, 15, seed, PlannerTypeSmallRoom)
			assert.NoError(t, err, "シード %d で失敗", seed)
			assert.NotNil(t, plan)

			// プレイヤー位置が設定されていることを確認
			_, _, hasPlayer := plan.GetPlayerStartPosition()
			assert.True(t, hasPlayer, "シード %d でプレイヤー位置なし", seed)
		}
	})
}

// TestIsConnectivityError は接続性エラー判定をテストする
func TestIsConnectivityError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil エラー",
			err:      nil,
			expected: false,
		},
		{
			name:     "接続性エラー（直接）",
			err:      ErrConnectivity,
			expected: true,
		},
		{
			name:     "接続性エラー（計画検証エラー経由）",
			err:      fmt.Errorf("計画検証エラー: %w", ErrConnectivity),
			expected: true,
		},
		{
			name:     "プレイヤー配置エラー（直接）",
			err:      ErrPlayerPlacement,
			expected: true,
		},
		{
			name:     "プレイヤー配置エラー（計画検証エラー経由）",
			err:      fmt.Errorf("計画検証エラー: %w", ErrPlayerPlacement),
			expected: true,
		},
		{
			name:     "その他のエラー",
			err:      fmt.Errorf("MetaPlan構築エラー: 何らかの問題"),
			expected: false,
		},
		{
			name:     "計画検証エラー",
			err:      fmt.Errorf("計画検証エラー: 何らかの検証失敗"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isConnectivityError(tc.err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
