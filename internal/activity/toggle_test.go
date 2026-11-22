package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToggleBehavior_持っている物のスイッチを点け消しする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	lamp := world.ECS.NewEntity()
	world.Components.Switch.Add(lamp, &gc.Switch{})
	world.Components.LocationInBackpack.Add(lamp, &gc.LocationInBackpack{Owner: actor})

	_, err := Execute(NewToggleActivity(lamp), actor, world)
	require.NoError(t, err)
	assert.True(t, world.Components.Switch.Get(lamp).On, "1回目で点く")
	assert.True(t, world.Components.StatsChanged.Has(actor), "持ち主の灯りを写し直させる")
	assert.True(t, query.GetVisionState(world).ConsumePendingUpdate(), "視界を計算し直させる")

	_, err = Execute(NewToggleActivity(lamp), actor, world)
	require.NoError(t, err)
	assert.False(t, world.Components.Switch.Get(lamp).On, "2回目で消える")
}

func TestToggleBehavior_Validate(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	noSwitch := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(noSwitch, &gc.LocationInBackpack{Owner: actor})
	others := world.ECS.NewEntity()
	world.Components.Switch.Add(others, &gc.Switch{})
	world.Components.LocationInBackpack.Add(others, &gc.LocationInBackpack{Owner: world.ECS.NewEntity()})

	tb := &ToggleBehavior{}
	require.Error(t, tb.Validate(NewToggleActivity(noSwitch), actor, world), "スイッチを持たない物は点け消しできない")
	require.Error(t, tb.Validate(NewToggleActivity(others), actor, world), "ほかの者の物は点け消しできない")
}
