package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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

func TestFeedFuelMenuState_灯りを満タンにしたら閉じる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 0
	_, err = lifecycle.SpawnBackpackItem(world, "lamp_oil", 2)
	require.NoError(t, err)
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return "", false }
	st := &FeedFuelMenuState{sink: LightFuelSink(world, lantern)}
	require.NoError(t, st.OnStart(world))
	_, err = st.Update(world)
	require.NoError(t, err)

	tr, err := st.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, tr.Type)
	assert.Equal(t, 1500, world.Components.Charge.Get(lantern).Current)
}

func TestRefillChoices(t *testing.T) {
	t.Parallel()
	type row struct {
		Label    string
		Disabled bool
	}
	tests := []struct {
		name  string
		setup func(world w.World) []row
	}{
		{name: "満たせる物が無ければ無効の行と閉じるを出す", setup: func(world w.World) []row {
			return []row{{Label: query.T(world, "Nothing to refill"), Disabled: true}, {Label: query.T(world, "Close")}}
		}},
		{name: "満タンでない物を並べる", setup: func(world w.World) []row {
			lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
			require.NoError(t, err)
			world.Components.Charge.Get(lantern).Current = 0
			return []row{{Label: query.GetEntityName(lantern, world)}, {Label: query.T(world, "Close")}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			want := tt.setup(world)

			title, choices := refillChoices(world)
			got := make([]row, 0, len(choices))
			for _, c := range choices {
				got = append(got, row{Label: c.Label, Disabled: c.Disabled})
			}
			assert.Equal(t, query.T(world, "Refill"), title)
			assert.Equal(t, want, got)
		})
	}
}

func TestLightFuelSink_満タンなら燃料を使わずログだけ出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	sink := LightFuelSink(world, lantern)
	assert.Equal(t, consts.Turn(0).String(), sink.gain(world, oil), "満タンなら増える量は0")

	require.NoError(t, sink.feed(world, oil))
	assert.True(t, world.ECS.Alive(oil), "燃料は使わない")
	assert.Equal(t, query.T(world, "It is already full."), query.GetGameLog(world).GetRecentEntries(1)[0].Text())
}
