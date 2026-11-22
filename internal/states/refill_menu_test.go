package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefillableItems_持っている満たせる物のうち満タンでない物を並べる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	used, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(used).Current = 100
	_, err = lifecycle.SpawnBackpackItem(world, "flashlight", 1)
	require.NoError(t, err)
	floor, err := lifecycle.SpawnFieldItem(world, "hand_lantern", 1, 1, 1)
	require.NoError(t, err)
	world.Components.Charge.Get(floor).Current = 100

	assert.Equal(t, []ecs.Entity{used}, refillableItems(world), "満タンの懐中電灯と床のランタンは並べない")
}

func TestLightFuelSink_同じ種別の燃料だけをくべられる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	sink := LightFuelSink(world, lantern)
	tests := []struct {
		rawID string
		want  bool
	}{
		{rawID: "lamp_oil", want: true},
		{rawID: "battery", want: false},
		{rawID: "firewood", want: false},
	}
	for _, tt := range tests {
		fuel, err := lifecycle.SpawnBackpackItem(world, tt.rawID, 1)
		require.NoError(t, err)
		assert.Equal(t, tt.want, sink.accepts(world, fuel), tt.rawID)
	}
}

func TestLightFuelSink_満タンになったら閉じてよい(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 0
	oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	sink := LightFuelSink(world, lantern)
	require.False(t, sink.done(world))

	require.NoError(t, sink.feed(world, oil))
	assert.True(t, sink.done(world), "1つで満タンになる")
}

func TestExecUseItem_スイッチを持つ物は点け消しする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	require.True(t, acceptUse(world, lantern), "スイッチを持つ物は使うの対象になる")

	_, err = execUseItem(world, lantern)
	require.NoError(t, err)
	assert.True(t, world.Components.Switch.Get(lantern).On)
	assert.True(t, world.ECS.Alive(lantern), "点け消しは物を消費しない")
}
