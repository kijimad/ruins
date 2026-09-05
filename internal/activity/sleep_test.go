package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSleepBehavior_疲れていないと眠れない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 0, Max: 2000})

	result, err := Execute(NewSleepActivity(), actor, world)
	require.NoError(t, err, "入眠拒否はユーザエラーで err にはならない")
	assert.False(t, result.Success, "快調では入眠できない")
	assert.False(t, world.Components.Sleeping.Has(actor), "Sleeping は付かない")
}

func TestSleepBehavior_入眠するとSleepingが付く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	// 疲労段階を Tired にして眠れるようにする
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1200, Max: 2000})

	result, err := Execute(NewSleepActivity(), actor, world)
	require.NoError(t, err)
	require.True(t, result.Success, "入眠できる")
	assert.True(t, world.Components.Sleeping.Has(actor), "入眠で Sleeping が付く")
}

func TestSleepBehavior_足元の寝具品質を写す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	world.Components.Fatigue.Add(actor, &gc.Fatigue{Current: 1200, Max: 2000})

	bed := world.ECS.NewEntity()
	world.Components.GridElement.Add(bed, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
	world.Components.Bedding.Add(bed, &gc.Bedding{Quality: 150})

	_, err := Execute(NewSleepActivity(), actor, world)
	require.NoError(t, err)
	assert.Equal(t, consts.Percent(150), world.Components.Sleeping.Get(actor).Quality,
		"足元の寝具の Quality を写す")
}
