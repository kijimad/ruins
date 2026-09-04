package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlannerChain_takeSnapshot_Recordingが無効なら記録しない(t *testing.T) {
	t.Parallel()

	chain := &PlannerChain{Recording: false}
	chain.takeSnapshot("Initial")

	assert.Empty(t, chain.Snapshots)
}

func TestPlannerChain_takeSnapshot_Recordingが有効ならラベル付きで記録する(t *testing.T) {
	t.Parallel()

	chain := &PlannerChain{
		Recording: true,
		PlanData: MetaPlan{
			Corridors: [][]gc.TileIdx{{1, 2, 3}},
		},
	}
	chain.takeSnapshot("Initial")

	require.Len(t, chain.Snapshots, 1)
	assert.Equal(t, "Initial", chain.Snapshots[0].Label)
	assert.Equal(t, [][]gc.TileIdx{{1, 2, 3}}, chain.Snapshots[0].Corridors)
}

func TestPlannerChain_takeSnapshot_記録後のPlanData変更はスナップショットに影響しない(t *testing.T) {
	t.Parallel()

	chain := &PlannerChain{
		Recording: true,
		PlanData: MetaPlan{
			Corridors: [][]gc.TileIdx{{1, 2, 3}},
		},
	}
	chain.takeSnapshot("Initial")

	// スナップショット取得後にPlanDataの内側スライスを書き換える
	chain.PlanData.Corridors[0][0] = 99

	require.Len(t, chain.Snapshots, 1)
	assert.Equal(t, [][]gc.TileIdx{{1, 2, 3}}, chain.Snapshots[0].Corridors,
		"深くコピーされていれば元データの変更はスナップショットに反映されないはず")
}

func TestDeepCloneCorridors_内側スライスも独立してコピーする(t *testing.T) {
	t.Parallel()

	src := [][]gc.TileIdx{{1, 2}, {3, 4, 5}}
	dst := deepCloneCorridors(src)

	assert.Equal(t, src, dst)

	dst[0][0] = 999
	assert.Equal(t, gc.TileIdx(1), src[0][0], "コピー先の変更が元のスライスに波及してはならない")
}

func TestDeepCloneCorridors_空スライスはそのまま空を返す(t *testing.T) {
	t.Parallel()

	dst := deepCloneCorridors([][]gc.TileIdx{})
	assert.Empty(t, dst)
}
