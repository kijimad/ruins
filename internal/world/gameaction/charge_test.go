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

func TestConsumeCharge_燃料が尽きると同じ置き場所の空の物に変わる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	battery, err := lifecycle.SpawnBackpackItem(world, "battery", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(battery).Current = 1

	ConsumeCharge(world, battery, 1)

	assert.False(t, world.ECS.Alive(battery))
	_, ok := query.FindInBackpack(world, player, "empty_battery")
	assert.True(t, ok, "空の物が持ち物に残る")
	assert.Equal(t, query.T(world, "%s is used up.", "Battery"), query.GetGameLog(world).GetRecentEntries(1)[0].Text())
}

func TestTransferCharge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		light     int
		fuel      int
		wantErr   error
		wantMoved int
		wantLight int
		wantFuel  int
		wantEmpty bool
	}{
		{name: "灯りの不足分だけ移り、燃料は残りを持つ", light: 1000, fuel: 1500, wantMoved: 500, wantLight: 1500, wantFuel: 1000},
		{name: "燃料の残りだけ移り、燃料は空の缶になる", light: 0, fuel: 600, wantMoved: 600, wantLight: 600, wantEmpty: true},
		{name: "満タンの灯りは満たさない", light: 1500, fuel: 1500, wantErr: ErrChargeFull, wantLight: 1500, wantFuel: 1500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
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
			_, hasCan := query.FindInBackpack(world, player, "empty_oil_can")
			assert.Equal(t, tt.wantEmpty, hasCan)
			assert.Equal(t, !tt.wantEmpty, world.ECS.Alive(oil))
			if !tt.wantEmpty {
				assert.Equal(t, tt.wantFuel, world.Components.Charge.Get(oil).Current)
			}
		})
	}
}

func TestTransferCharge_種別の合わない燃料では満たさない(t *testing.T) {
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

func TestDrainCharge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		light     int
		lightMax  int
		cans      int
		wantErr   error
		wantMoved int
		wantLight int
	}{
		{name: "空の缶が灯りの残量を持った灯油に戻る", light: 600, lightMax: 1500, cans: 1, wantMoved: 600, wantLight: 0},
		{name: "燃料の上限を超える分は灯りに残る", light: 2000, lightMax: 2000, cans: 1, wantMoved: 1500, wantLight: 500},
		{name: "空の缶を持たなければ抜けない", light: 600, lightMax: 1500, wantErr: ErrNoContainer, wantLight: 600},
		{name: "残量0の灯りは抜けない", light: 0, lightMax: 1500, cans: 1, wantErr: ErrChargeEmpty, wantLight: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			lantern, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
			require.NoError(t, err)
			charge := world.Components.Charge.Get(lantern)
			charge.Max, charge.Current = tt.lightMax, tt.light
			if tt.cans > 0 {
				_, err = lifecycle.SpawnBackpackItem(world, "empty_oil_can", tt.cans)
				require.NoError(t, err)
			}

			oil, moved, err := DrainCharge(world, lantern)

			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantMoved, moved)
			assert.Equal(t, tt.wantLight, world.Components.Charge.Get(lantern).Current)
			if tt.wantErr != nil {
				return
			}
			assert.Equal(t, player, world.Components.LocationInBackpack.Get(oil).Owner, "入れ物と同じ持ち物の中にできる")
			assert.Equal(t, "lamp_oil", world.Components.RawID.Get(oil).ID)
			assert.Equal(t, tt.wantMoved, world.Components.Charge.Get(oil).Current)
			_, hasCan := query.FindInBackpack(world, player, "empty_oil_can")
			assert.False(t, hasCan, "入れ物を使う")
		})
	}
}

func TestDrainCharge_灯りが0になってもログを出さず視界を計算し直させる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	flashlight, err := lifecycle.SpawnBackpackItem(world, "flashlight", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(flashlight).Current = 300
	_, err = lifecycle.SpawnBackpackItem(world, "empty_battery", 1)
	require.NoError(t, err)
	world.Components.StatsChanged.Remove(player)
	query.GetVisionState(world).ConsumePendingUpdate()
	logs := query.GetGameLog(world).Count()

	_, _, err = DrainCharge(world, flashlight)

	require.NoError(t, err)
	assert.Equal(t, 0, world.Components.Charge.Get(flashlight).Current)
	assert.Equal(t, logs, query.GetGameLog(world).Count())
	assert.True(t, world.Components.StatsChanged.Has(player))
	assert.True(t, query.GetVisionState(world).ConsumePendingUpdate())
}

func TestTransferChargeとDrainCharge_繰り返しても燃料と空の物の数の合計は変わらない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	flashlight, err := lifecycle.SpawnBackpackItem(world, "flashlight", 1)
	require.NoError(t, err)
	world.Components.Charge.Get(flashlight).Current = 0
	_, err = lifecycle.SpawnBackpackItem(world, "battery", 1)
	require.NoError(t, err)

	for range 3 {
		battery, ok := query.FindInBackpack(world, player, "battery")
		require.True(t, ok)
		_, err := TransferCharge(world, battery, flashlight)
		require.NoError(t, err)
		_, _, err = DrainCharge(world, flashlight)
		require.NoError(t, err)
	}

	count := map[string]int{}
	for _, stack := range query.BackpackStacks(world, player) {
		if id := world.Components.RawID.Get(stack.Rep).ID; id == "battery" || id == "empty_battery" {
			count[id] += stack.Count
		}
	}
	assert.Equal(t, map[string]int{"battery": 1}, count)
}
