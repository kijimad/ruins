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

func TestConsumeCharge_燃料は尽きても残量0の入れ物として残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	battery, err := lifecycle.SpawnBackpackItem(world, "battery", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(battery).Current = 1

	ConsumeCharge(world, battery, 1)

	require.True(t, world.ECS.Alive(battery))
	assert.Equal(t, 0, world.Components.Charge.Get(battery).Current)
	assert.Equal(t, query.T(world, "%s ran out.", "Battery"), query.GetGameLog(world).GetRecentEntries(1)[0].Text())
}

func TestTransferCharge_燃料から灯りへ満たす(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		light     int
		fuel      int
		wantErr   error
		wantMoved int
		wantLight int
		wantFuel  int
	}{
		{name: "灯りの空きだけ移り、燃料は残りを持つ", light: 1000, fuel: 1500, wantMoved: 500, wantLight: 1500, wantFuel: 1000},
		{name: "燃料の残りだけ移り、燃料は残量0で残る", light: 0, fuel: 600, wantMoved: 600, wantLight: 600, wantFuel: 0},
		{name: "満タンの灯りは満たさない", light: 1500, fuel: 1500, wantErr: ErrChargeFull, wantLight: 1500, wantFuel: 1500},
		{name: "空の燃料からは満たさない", light: 0, fuel: 0, wantErr: ErrChargeEmpty, wantLight: 0, wantFuel: 0},
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
			oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
			require.NoError(t, err)
			world.Components.Charge.Get(oil).Current = tt.fuel

			moved, err := TransferCharge(world, oil, lantern)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantMoved, moved)
			assert.Equal(t, tt.wantLight, world.Components.Charge.Get(lantern).Current)
			assert.Equal(t, tt.wantFuel, world.Components.Charge.Get(oil).Current)
		})
	}
}

func TestTransferCharge_灯りから燃料へ抜く(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		light         int
		lightMax      int
		container     int
		wantErr       error
		wantMoved     int
		wantLight     int
		wantContainer int
	}{
		{name: "空の缶へ灯りの残量が移る", light: 600, lightMax: 1500, container: 0, wantMoved: 600, wantLight: 0, wantContainer: 600},
		{name: "使いかけの缶へは空きだけ移る", light: 1000, lightMax: 1500, container: 1000, wantMoved: 500, wantLight: 500, wantContainer: 1500},
		{name: "缶の上限を超える分は灯りに残る", light: 2000, lightMax: 2000, container: 0, wantMoved: 1500, wantLight: 500, wantContainer: 1500},
		{name: "満タンの缶へは抜けない", light: 600, lightMax: 1500, container: 1500, wantErr: ErrChargeFull, wantLight: 600, wantContainer: 1500},
		{name: "残量0の灯りからは抜けない", light: 0, lightMax: 1500, container: 0, wantErr: ErrChargeEmpty, wantLight: 0, wantContainer: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
			require.NoError(t, err)
			charge := world.Components.Charge.Get(lantern)
			charge.Max, charge.Current = tt.lightMax, tt.light
			oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
			require.NoError(t, err)
			world.Components.Charge.Get(oil).Current = tt.container

			moved, err := TransferCharge(world, lantern, oil)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantMoved, moved)
			assert.Equal(t, tt.wantLight, world.Components.Charge.Get(lantern).Current)
			assert.Equal(t, tt.wantContainer, world.Components.Charge.Get(oil).Current)
		})
	}
}

func TestTransferCharge_種別の合わない物の間では移さない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 0
	battery, err := lifecycle.SpawnBackpackItem(world, "battery", 1)
	require.NoError(t, err)

	_, err = TransferCharge(world, battery, lantern)

	require.ErrorIs(t, err, ErrRefillMismatch)
	assert.Equal(t, 0, world.Components.Charge.Get(lantern).Current)
	assert.Equal(t, 1000, world.Components.Charge.Get(battery).Current, "燃料は使わない")
}

func TestTransferCharge_灯りが0から戻ると持ち主に写し直させる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(lantern).Current = 0
	oil, err := lifecycle.SpawnBackpackItem(world, "lamp_oil", 1)
	require.NoError(t, err)
	world.Components.StatsChanged.Remove(player)
	query.GetVisionState(world).ConsumePendingUpdate()

	_, err = TransferCharge(world, oil, lantern)

	require.NoError(t, err)
	assert.True(t, world.Components.StatsChanged.Has(player))
	assert.True(t, query.GetVisionState(world).ConsumePendingUpdate())
}

func TestTransferCharge_灯りが0になってもログを出さず視界を計算し直させる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	flashlight, err := lifecycle.SpawnBackpackItem(world, "flashlight", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(flashlight).Current = 300
	battery, err := lifecycle.SpawnBackpackItem(world, "battery", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(battery).Current = 0
	world.Components.StatsChanged.Remove(player)
	query.GetVisionState(world).ConsumePendingUpdate()
	logs := query.GetGameLog(world).Count()

	_, err = TransferCharge(world, flashlight, battery)

	require.NoError(t, err)
	assert.Equal(t, 0, world.Components.Charge.Get(flashlight).Current)
	assert.Equal(t, logs, query.GetGameLog(world).Count())
	assert.True(t, world.Components.StatsChanged.Has(player))
	assert.True(t, query.GetVisionState(world).ConsumePendingUpdate())
}

func TestTransferCharge_束の1個から移すとその1個だけ別の束に分かれる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	flashlight, err := lifecycle.SpawnBackpackItem(world, "flashlight", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(flashlight).Current = 600
	battery, err := lifecycle.SpawnBackpackItem(world, "battery", 2)
	require.NoError(t, err)
	members := query.StackMembers(world, battery)
	require.Len(t, members, 2)

	_, err = TransferCharge(world, members[0], flashlight)

	require.NoError(t, err)
	assert.Equal(t, 600, world.Components.Charge.Get(members[0]).Current)
	assert.Equal(t, 1000, world.Components.Charge.Get(members[1]).Current, "同じ束のほかの電池は変わらない")
	var counts []int
	for _, stack := range query.BackpackStacks(world, player) {
		if world.Components.RawID.Get(stack.Rep).ID == "battery" {
			counts = append(counts, stack.Count)
		}
	}
	assert.Equal(t, []int{1, 1}, counts)
}

func TestTransferCharge_満たして抜くを繰り返しても残量の合計は変わらない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	flashlight, err := lifecycle.SpawnBackpackItem(world, "flashlight", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(flashlight).Current = 300
	battery, err := lifecycle.SpawnBackpackItem(world, "battery", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(battery).Current = 400

	for range 3 {
		_, err := TransferCharge(world, battery, flashlight)
		require.NoError(t, err)
		_, err = TransferCharge(world, flashlight, battery)
		require.NoError(t, err)
	}

	assert.Equal(t, 700, world.Components.Charge.Get(flashlight).Current+world.Components.Charge.Get(battery).Current)
}
