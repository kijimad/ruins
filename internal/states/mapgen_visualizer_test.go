package states_test

import (
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/states"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapGenVisualizerState_開始した後に止めるとスポーンした物を消す(t *testing.T) {
	t.Parallel()
	plannerTypes := collectPlannerTypes(t)
	i := slices.IndexFunc(plannerTypes, func(pt mapplanner.PlannerType) bool { return pt.Name == "small_room" })
	require.GreaterOrEqual(t, i, 0)

	world := testutil.InitTestWorld(t)
	gridCount := func() int {
		q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
		n := q.Count()
		q.Close()
		return n
	}
	before := gridCount()
	st := &gs.MapGenVisualizerState{PlannerType: plannerTypes[i], Seed: 1}
	require.NoError(t, st.OnStart(world))
	require.Greater(t, gridCount(), before, "開始するとタイルなどをスポーンする")
	require.NoError(t, st.OnStop(world))
	assert.Equal(t, before, gridCount(), "止めるとスポーンした物が残らない")
}
