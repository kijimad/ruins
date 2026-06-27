package mapplanner_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// TestGolden_Snapshot は各プランナータイプの全フェーズのスナップショットをJSONゴールデンテストで検証する。
// 固定シードで再現性を保証し、マップ生成アルゴリズム変更時のリグレッションを検知する
func TestGolden_Snapshot(t *testing.T) {
	t.Parallel()

	plannerTypes := []mapplanner.PlannerType{
		mapplanner.PlannerTypeSmallRoom,
		mapplanner.PlannerTypeBigRoom,
		mapplanner.PlannerTypeCave,
		mapplanner.PlannerTypeRuins,
		mapplanner.PlannerTypeForest,
	}
	seed := uint64(12345)

	for _, pt := range plannerTypes {
		chain, err := pt.PlannerFunc(consts.MapTileWidth, consts.MapTileHeight, seed)
		require.NoError(t, err)
		chain.Recording = true
		chain.PlanData.RawMaster = mapplanner.CreateTestRawMaster()
		require.NoError(t, chain.Plan())

		for i, snap := range chain.Snapshots {
			t.Run(fmt.Sprintf("%s/Phase%d_%s", pt.Name, i, snap.Label), func(t *testing.T) {
				t.Parallel()
				data, err := json.MarshalIndent(snap, "", "  ")
				require.NoError(t, err)

				g := goldie.New(t, goldie.WithNameSuffix(".json"))
				g.Assert(t, t.Name(), data)
			})
		}
	}
}
