package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumeCharge_尽きる前は残り視界を計算し直させない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	torch, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	query.GetVisionState(world).ConsumePendingUpdate()

	ConsumeCharge(world, torch, 1)

	require.True(t, world.ECS.Alive(torch))
	assert.Equal(t, 599, world.Components.Charge.Get(torch).Current)
	assert.False(t, query.GetVisionState(world).ConsumePendingUpdate())
}

func TestConsumeCharge_残量を超えて減らしても尽きて消える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	torch, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)

	ConsumeCharge(world, torch, 1000)

	assert.False(t, world.ECS.Alive(torch))
	assert.True(t, query.GetVisionState(world).ConsumePendingUpdate())
}

func TestConsumeCharge_ほかの者の灯りが尽きてもログを出さない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	enemy := world.ECS.NewEntity()
	torch, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	lifecycle.MoveToEquip(world, torch, enemy, gc.SlotHand1)
	before := query.GetGameLog(world).Count()

	ConsumeCharge(world, torch, 600)

	assert.False(t, world.ECS.Alive(torch))
	assert.Equal(t, before, query.GetGameLog(world).Count())
}

func TestConsumeCharge_減っていく途中はログを出さない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	torch, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	lifecycle.MoveToEquip(world, torch, player, gc.SlotHand2)
	world.Components.Charge.Get(torch).Current = 300
	logs := query.GetGameLog(world).Count()

	ConsumeCharge(world, torch, 1)

	assert.Equal(t, logs, query.GetGameLog(world).Count(), "減ってもログは出さない")
}

func TestConsumeCharge_プレイヤーの物が尽きるとログを出す(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		rawID string
		place func(world w.World, item, player ecs.Entity) error
		want  i18n.Msgid
	}{
		{name: "手持ち欄の灯りは使い切ると消える", rawID: "torch", place: func(world w.World, item, player ecs.Entity) error {
			lifecycle.MoveToEquip(world, item, player, gc.SlotHand2)
			return nil
		}, want: "%s is used up."},
		{name: "バックパックの灯りは使い切ると消える", rawID: "torch", place: lifecycle.MoveToBackpack, want: "%s is used up."},
		{name: "手持ち欄の武器も同じ文で消える", rawID: "iron_sword", place: func(world w.World, item, player ecs.Entity) error {
			lifecycle.MoveToEquip(world, item, player, gc.SlotHand1)
			return nil
		}, want: "%s is used up."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			item, err := lifecycle.SpawnFieldItem(world, tt.rawID, 3, 4, 1)
			require.NoError(t, err)
			require.NoError(t, tt.place(world, item, player))
			world.Components.Charge.Get(item).Current = 1
			name := query.GetEntityName(item, world)
			logs := query.GetGameLog(world).Count()

			ConsumeCharge(world, item, 1)

			assert.False(t, world.ECS.Alive(item))
			require.Equal(t, logs+1, query.GetGameLog(world).Count())
			assert.Equal(t, query.T(world, tt.want, name), query.GetGameLog(world).GetRecentEntries(1)[0].Text())
		})
	}
}

func TestConsumeCharge_満たせる灯りは尽きても残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 1
	world.Components.StatsChanged.Remove(player)

	ConsumeCharge(world, lantern, 1)

	require.True(t, world.ECS.Alive(lantern), "満たせる物は消えない")
	assert.Equal(t, 0, world.Components.Charge.Get(lantern).Current)
	assert.True(t, world.Components.StatsChanged.Has(player), "持ち主の灯りを写し直させる")
	assert.Equal(t, query.T(world, "%s ran out.", "Hand Lantern"), query.GetGameLog(world).GetRecentEntries(1)[0].Text())
}

func TestRefillCharge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		current     int
		wantErr     error
		wantCharge  int
		wantFuelHas bool
	}{
		{name: "減った灯りは燃料を1つ使って満タンに戻る", current: 0, wantCharge: 1500, wantFuelHas: false},
		{name: "満タンなら燃料を使わない", current: 1500, wantErr: ErrChargeFull, wantCharge: 1500, wantFuelHas: true},
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
			oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
			require.NoError(t, err)

			err = RefillCharge(world, lantern, oil)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantCharge, world.Components.Charge.Get(lantern).Current)
			assert.Equal(t, tt.wantFuelHas, world.ECS.Alive(oil))
		})
	}
}

func TestRefillCharge_種別の合わない燃料では満たさない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 0
	battery, err := lifecycle.SpawnBackpackItem(world, "battery", 1)
	require.NoError(t, err)

	require.ErrorIs(t, RefillCharge(world, lantern, battery), ErrRefillMismatch)
	assert.Equal(t, 0, world.Components.Charge.Get(lantern).Current)
	assert.True(t, world.ECS.Alive(battery), "燃料は使わない")
}
