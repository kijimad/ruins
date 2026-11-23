package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
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

func TestLightDrainSink_抜く先が満タンになって無くなれば閉じてよい(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(oil).Current = 1125
	sink := LightDrainSink(world, lantern)

	require.NoError(t, sink.feed(world, oil))

	assert.Equal(t, 1125, world.Components.Charge.Get(lantern).Current, "灯りには抜ききれない分が残る")
	assert.True(t, sink.done(world))
}

func TestRefillMenuState_Fetch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		light         int
		fuel          int
		fillDisabled  bool
		drainDisabled bool
	}{
		{name: "燃料を持たなければどちらも選べない", light: 600, fuel: -1, fillDisabled: true, drainDisabled: true},
		{name: "減っていて使いかけの燃料を持てばどちらも選べる", light: 600, fuel: 700},
		{name: "満タンなら満たせず満タンでない燃料へ抜ける", light: 1500, fuel: 700, fillDisabled: true},
		{name: "空の燃料では満たせないが抜ける", light: 600, fuel: 0, fillDisabled: true},
		{name: "満タンの燃料へは抜けない", light: 600, fuel: 1500, drainDisabled: true},
		{name: "残量0の灯りからは抜けない", light: 0, fuel: 0, fillDisabled: true, drainDisabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
			require.NoError(t, err)
			world.Components.Charge.Get(lantern).Current = tt.light
			if tt.fuel >= 0 {
				oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
				require.NoError(t, err)
				world.Components.Charge.Get(oil).Current = tt.fuel
			}
			expected := []refillRow{{Entity: lantern, Disabled: tt.fillDisabled}, {Entity: lantern, Disabled: tt.drainDisabled}}

			props, err := (&RefillMenuState{}).Fetch(world)
			require.NoError(t, err)

			labels := make([]string, 0, len(props.Tabs))
			got := make([]refillRow, 0, len(props.Tabs))
			for _, tab := range props.Tabs {
				labels = append(labels, tab.Label)
				require.Len(t, tab.Rows, 1)
				got = append(got, tab.Rows[0])
			}
			assert.Equal(t, []string{query.T(world, "Fill"), query.T(world, "Drain")}, labels)
			assert.Equal(t, expected, got)
		})
	}
}

func TestRefillMenuState_灯りを持たなければどのタブも空(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	props, err := (&RefillMenuState{}).Fetch(world)

	require.NoError(t, err)
	require.Len(t, props.Tabs, 2)
	assert.Empty(t, props.Tabs[0].Rows)
	assert.Empty(t, props.Tabs[1].Rows)
}

func TestRefillMenuState_選べる灯りを選ぶと給油メニューへ進み選べない灯りでは進まない(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		tab   int
		light int
		fuel  int
		want  es.TransType
	}{
		{name: "満たせる灯り", tab: 0, light: 600, fuel: 700, want: es.TransPush},
		{name: "残量のある燃料が無ければ満たせない", tab: 0, light: 600, fuel: 0, want: es.TransNone},
		{name: "満タンの灯りは満たせない", tab: 0, light: 1500, fuel: 700, want: es.TransNone},
		{name: "抜ける灯り", tab: 1, light: 600, fuel: 700, want: es.TransPush},
		{name: "残量0の灯りからは抜けない", tab: 1, light: 0, fuel: 700, want: es.TransNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t, testutil.WithUI())
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
			require.NoError(t, err)
			world.Components.Charge.Get(lantern).Current = tt.light
			oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
			require.NoError(t, err)
			world.Components.Charge.Get(oil).Current = tt.fuel
			world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return "", false }
			st := &RefillMenuState{}
			require.NoError(t, st.OnStart(world))
			_, err = st.Update(world)
			require.NoError(t, err)
			st.screen.SetTab(tt.tab)
			_, err = st.Update(world)
			require.NoError(t, err)

			tr, err := st.DoAction(world, inputmapper.ActionMenuSelect)

			require.NoError(t, err)
			assert.Equal(t, tt.want, tr.Type)
		})
	}
}

func TestRefillMenuState_同じ世界から作ったpropsは等しい(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	st := &RefillMenuState{}

	first, err := st.Fetch(world)
	require.NoError(t, err)
	second, err := st.Fetch(world)
	require.NoError(t, err)

	assert.Equal(t, first, second, "props が関数を持つと毎フレーム変わったと判定され、描き直しが止まらない")
}

func TestRefillMenuState_上の灯りが選べなければ下の灯りにカーソルが止まり選ぶと進む(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	// 名前の順で Flashlight が上、Hand Lantern が下に並ぶ。満タンの懐中電灯は満たせない
	_, err = lifecycle.SpawnBackpackItem(world, "flashlight", 1)
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 600
	_, err = lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return "", false }
	st := &RefillMenuState{}
	require.NoError(t, st.OnStart(world))
	_, err = st.Update(world)
	require.NoError(t, err)

	got, ok := st.selectedEntity()
	require.True(t, ok)
	assert.Equal(t, lantern, got)
	tr, err := st.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransPush, tr.Type)
}

func TestRefillMenuState_すべての行が選べなくても詳細はカーソルの灯りを見せる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t, testutil.WithUI())
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Resources.InputSource = func() (inputmapper.ActionID, bool) { return "", false }
	st := &RefillMenuState{}
	require.NoError(t, st.OnStart(world))
	_, err = st.Update(world)
	require.NoError(t, err)

	got, ok := st.selectedEntity()
	require.True(t, ok)
	assert.Equal(t, lantern, got, "燃料を持たないので行は選べないが、詳細の対象はカーソルの灯り")
	tr, err := st.DoAction(world, inputmapper.ActionMenuSelect)
	require.NoError(t, err)
	assert.Equal(t, es.TransNone, tr.Type)
}
