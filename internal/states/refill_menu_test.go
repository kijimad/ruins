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

func TestRefillableItems_持っている燃料を受ける灯りを満タンの物も含めて並べる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	used, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(used).Current = 100
	full, err := lifecycle.SpawnBackpackItem(world, "flashlight", 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "torch", 1)
	require.NoError(t, err)
	floor, err := lifecycle.SpawnFieldItem(world, "hand_lantern", 1, 1, 1)
	require.NoError(t, err)
	world.Components.Charge.Get(floor).Current = 100

	assert.ElementsMatch(t, []ecs.Entity{used, full}, refillableItems(world), "使い切りの松明と床のランタンは並べない")
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
	assert.True(t, sink.done(world), "満タンの燃料1つで満タンになる")
}

func TestLightFuelSink_燃料の残りが足りなければ満タンにならず閉じない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 0
	oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(oil).Current = 400
	sink := LightFuelSink(world, lantern)
	assert.Equal(t, consts.Turn(400).String(), sink.gain(world, oil), "増える量は燃料の残り")

	require.NoError(t, sink.feed(world, oil))
	assert.Equal(t, 400, world.Components.Charge.Get(lantern).Current)
	assert.False(t, sink.done(world))
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
		{name: "満タンの物も並べる", setup: func(world w.World) []row {
			lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
			require.NoError(t, err)
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

func TestLightRefillChoices(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		current   int
		fuel      int
		wantFill  bool
		wantDrain bool
	}{
		{name: "燃料を持たなければどちらも選べない", current: 600, fuel: -1},
		{name: "減っていて残量のある燃料を持てば満たせ、抜ける", current: 600, fuel: 700, wantFill: true, wantDrain: true},
		{name: "満タンなら満たせず、満タンでない燃料へ抜ける", current: 1500, fuel: 700, wantDrain: true},
		{name: "空の燃料では満たせないが、抜ける", current: 600, fuel: 0, wantDrain: true},
		{name: "満タンの燃料へは抜けない", current: 600, fuel: 1500, wantFill: true},
		{name: "残量0の灯りからは抜けない", current: 0, fuel: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
			require.NoError(t, err)
			world.Components.Charge.Get(lantern).Current = tt.current
			if tt.fuel >= 0 {
				oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
				require.NoError(t, err)
				world.Components.Charge.Get(oil).Current = tt.fuel
			}

			title, choices := lightRefillChoices(world, lantern)

			assert.Equal(t, query.GetEntityName(lantern, world), title)
			labels := make([]string, 0, len(choices))
			enabled := make([]bool, 0, len(choices))
			for _, c := range choices {
				labels = append(labels, c.Label)
				enabled = append(enabled, !c.Disabled)
			}
			assert.Equal(t, []string{query.T(world, "Fill"), query.T(world, "Drain"), query.T(world, "Close")}, labels)
			assert.Equal(t, []bool{tt.wantFill, tt.wantDrain, true}, enabled)
		})
	}
}

func TestLightDrainSink_空の燃料へ抜いてログを出し灯りが0になれば閉じてよい(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 600
	oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(oil).Current = 0
	sink := LightDrainSink(world, lantern)
	require.True(t, sink.accepts(world, oil))
	assert.Equal(t, consts.Turn(600).String(), sink.gain(world, oil), "抜ける量は灯りの残量")

	require.NoError(t, sink.feed(world, oil))

	assert.Equal(t, 600, world.Components.Charge.Get(oil).Current)
	assert.True(t, sink.done(world))
	assert.Equal(t, query.T(world, "Drained %s into %s.", "Hand Lantern", "Lamp Oil"), query.GetGameLog(world).GetRecentEntries(1)[0].Text())
}
