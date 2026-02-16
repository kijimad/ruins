package mapplanner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
)

func TestForestPlanner(t *testing.T) {
	t.Parallel()

	t.Run("ForestPlannerが正常に作成される", func(t *testing.T) {
		t.Parallel()
		chain, err := NewForestPlanner(30, 30, 12345)
		require.NoError(t, err)
		assert.NotNil(t, chain)
		assert.NotNil(t, chain.Starter)
	})

	t.Run("ForestPlannerでマップを生成", func(t *testing.T) {
		t.Parallel()
		chain, err := NewForestPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		// 空き地が生成されていることを確認
		assert.NotEmpty(t, chain.PlanData.Rooms, "空き地が生成されていない")

		// 床タイルと壁タイルの両方が存在することを確認
		floorCount := 0
		wallCount := 0
		for _, tile := range chain.PlanData.Tiles {
			if !tile.BlockPass {
				floorCount++
			} else {
				wallCount++
			}
		}
		assert.Greater(t, floorCount, 0, "床タイルが存在しない")
		assert.Greater(t, wallCount, 0, "壁タイルが存在しない")
	})

	t.Run("生成された空き地が有効な範囲内にある", func(t *testing.T) {
		t.Parallel()
		width, height := gc.Tile(30), gc.Tile(30)
		chain, err := NewForestPlanner(width, height, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		actualWidth := chain.PlanData.Level.TileWidth
		actualHeight := chain.PlanData.Level.TileHeight

		for i, room := range chain.PlanData.Rooms {
			assert.GreaterOrEqual(t, int(room.X1), 0, "空き地%dのX1が負の値", i)
			assert.GreaterOrEqual(t, int(room.Y1), 0, "空き地%dのY1が負の値", i)
			assert.LessOrEqual(t, int(room.X2), int(actualWidth), "空き地%dのX2が幅を超えている", i)
			assert.LessOrEqual(t, int(room.Y2), int(actualHeight), "空き地%dのY2が高さを超えている", i)
			assert.LessOrEqual(t, int(room.X1), int(room.X2), "空き地%dのX座標が逆転している", i)
			assert.LessOrEqual(t, int(room.Y1), int(room.Y2), "空き地%dのY座標が逆転している", i)
		}
	})

	t.Run("異なるサイズのマップで動作確認", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name   string
			width  gc.Tile
			height gc.Tile
		}{
			{"小さいマップ", 15, 15},
			{"中サイズマップ", 30, 30},
			{"大きいマップ", 50, 50},
			{"横長マップ", 40, 20},
			{"縦長マップ", 20, 40},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				chain, err := NewForestPlanner(tc.width, tc.height, 12345)
				require.NoError(t, err)
				chain.PlanData.RawMaster = CreateTestRawMaster()
				err = chain.Plan()
				require.NoError(t, err)

				actualWidth := chain.PlanData.Level.TileWidth
				actualHeight := chain.PlanData.Level.TileHeight
				expectedCount := int(actualWidth) * int(actualHeight)
				assert.Len(t, chain.PlanData.Tiles, expectedCount,
					"%sのタイル数が正しくない", tc.name)
				assert.Greater(t, len(chain.PlanData.Rooms), 0,
					"%sで空き地が生成されていない", tc.name)
			})
		}
	})

	t.Run("上端から下端への接続性が保たれる", func(t *testing.T) {
		t.Parallel()
		// 多数シードで接続性を確認し、縦通路で上下端への接続が保証されていることを検証する
		for seed := uint64(0); seed < 50; seed++ {
			chain, err := NewForestPlanner(30, 30, seed)
			require.NoError(t, err, "seed=%d", seed)
			chain.PlanData.RawMaster = CreateTestRawMaster()
			err = chain.Plan()
			require.NoError(t, err, "seed=%d", seed)

			pf := NewPathFinder(&chain.PlanData)
			err = pf.ValidateConnectivity()
			assert.NoError(t, err, "seed=%dで接続性検証に失敗した", seed)
		}
	})
}

func TestForestPlannerConnectivityIntegration(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.RawMaster = CreateTestRawMaster()

	for seed := uint64(0); seed < 20; seed++ {
		metaPlan, err := Plan(world, 50, 50, &seed, PlannerTypeForest)
		assert.NoError(t, err, "seed=%dで森プラン生成に失敗した", seed)
		if err != nil {
			continue
		}
		assert.NotNil(t, metaPlan, "seed=%dのMetaPlanがnil", seed)
	}
}
