package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreshnessStageOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		now  consts.Turn // 生成は TotalTurns=0。bread の ShelfLife は 1500
		want gc.FreshnessStage
	}{
		{"新鮮", 0, gc.FreshnessFresh},
		{"劣化", 1500, gc.FreshnessStale},
		{"腐敗", 3000, gc.FreshnessRotten},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			bread, err := lifecycle.SpawnFieldItem(world, "bread", 5, 5, 1)
			require.NoError(t, err)
			query.GetGameTime(world).TotalTurns = tt.now

			stage, ok := query.FreshnessStageOf(world, bread)
			require.True(t, ok, "bread は腐敗する")
			assert.Equal(t, tt.want, stage)
		})
	}

	t.Run("Perishableを持たない食料はok=false", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sword, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 5, 5, 1)
		require.NoError(t, err)

		_, ok := query.FreshnessStageOf(world, sword)
		assert.False(t, ok)
	})
}

func TestFreshnessMarker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		now     consts.Turn
		wantHas bool
	}{
		{"新鮮はマーカー無し", 0, false},
		{"劣化はマーカーあり", 1500, true},
		{"腐敗はマーカーあり", 3000, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			bread, err := lifecycle.SpawnFieldItem(world, "bread", 5, 5, 1)
			require.NoError(t, err)
			query.GetGameTime(world).TotalTurns = tt.now

			assert.Equal(t, tt.wantHas, query.FreshnessMarker(world, bread) != "")
		})
	}
}

func TestFreshnessLabel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Fresh", query.FreshnessLabel(gc.FreshnessFresh))
	assert.Equal(t, "Stale", query.FreshnessLabel(gc.FreshnessStale))
	assert.Equal(t, "Rotten", query.FreshnessLabel(gc.FreshnessRotten))
}
