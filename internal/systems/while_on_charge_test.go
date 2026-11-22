package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhileOnChargeSystem_床で点いた松明は燃え尽きて消える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	torch, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	world.Components.Charge.Get(torch).Current = 2
	sys := &WhileOnChargeSystem{}

	require.NoError(t, sys.Update(world))
	assert.Equal(t, 1, world.Components.Charge.Get(torch).Current, "照らしている間は1ターンに1減る")

	query.GetVisionState(world).ConsumePendingUpdate()
	logs := query.GetGameLog(world).Count()
	require.NoError(t, sys.Update(world))
	assert.False(t, world.ECS.Alive(torch), "尽きると消える")
	assert.Equal(t, logs, query.GetGameLog(world).Count(), "床の灯りが尽きてもログを出さない")
	assert.True(t, query.GetVisionState(world).ConsumePendingUpdate(), "消えると視界を計算し直させる")
}

func TestWhileOnChargeSystem_照らしている灯りだけが減る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	switchedOff, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	world.Components.LightSource.Get(switchedOff).Enabled = false
	carried, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	require.NoError(t, lifecycle.MoveToBackpack(world, carried, player))
	others, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	require.NoError(t, lifecycle.MoveToBackpack(world, others, world.ECS.NewEntity()))

	require.NoError(t, (&WhileOnChargeSystem{}).Update(world))
	assert.Equal(t, 600, world.Components.Charge.Get(switchedOff).Current, "スイッチを切った床の灯りは減らない")
	assert.Equal(t, 599, world.Components.Charge.Get(carried).Current, "プレイヤーのバックパックの松明は照らして減る")
	assert.Equal(t, 600, world.Components.Charge.Get(others).Current, "ほかの者のバックパックの松明は照らさず減らない")
}

func TestWhileOnChargeSystem_手持ち欄の灯りは2本とも燃え尽きる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	require.NoError(t, lifecycle.UnequipAll(world, player))

	var torches [2]ecs.Entity
	for i, slot := range []gc.EquipmentSlotNumber{gc.SlotHand1, gc.SlotHand2} {
		torch, err := lifecycle.SpawnBackpackItem(world, "torch", 1)
		require.NoError(t, err)
		lifecycle.MoveToEquip(world, torch, player, slot)
		world.Components.Charge.Get(torch).Current = 1
		torches[i] = torch
	}
	world.Components.StatsChanged.Remove(player)

	require.NoError(t, (&WhileOnChargeSystem{}).Update(world))
	for _, torch := range torches {
		assert.False(t, world.ECS.Alive(torch), "同じターンに2本とも燃え尽きる")
	}
	assert.True(t, world.Components.StatsChanged.Has(player), "装備の灯りを写し直させる")
	entries := query.GetGameLog(world).GetRecentEntries(1)
	require.Len(t, entries, 1)
	assert.Equal(t, query.T(world, "%s is used up.", "Torch"), entries[0].Text())
}

func TestWhileOnChargeSystem_束で床に置いた松明はどれも減る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	torch, err := lifecycle.SpawnBackpackItem(world, "torch", 3)
	require.NoError(t, err)
	torches := query.StackMembers(world, torch)
	require.Len(t, torches, 3)

	lifecycle.MoveStackToField(world, torch, consts.Coord[consts.Tile]{X: 6, Y: 5}, player)
	require.NoError(t, (&WhileOnChargeSystem{}).Update(world))

	for _, e := range torches {
		assert.Equal(t, 599, world.Components.Charge.Get(e).Current, "床に置いた束はどれも照らして減る")
	}
}

func TestWhileOnChargeSystem_切ったランタンは持っていても減らない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	off, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	on, err := lifecycle.SpawnBackpackItem(world, "hand_lantern", 1)
	require.NoError(t, err)
	world.Components.Switch.Get(on).On = true

	require.NoError(t, (&WhileOnChargeSystem{}).Update(world))
	assert.Equal(t, 1500, world.Components.Charge.Get(off).Current, "切ったランタンは減らない")
	assert.Equal(t, 1499, world.Components.Charge.Get(on).Current, "点けたランタンは減る")
}

func TestWhileOnChargeSystem_床に置いたオンのランタンは減る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	lantern, err := lifecycle.SpawnFieldItem(world, "hand_lantern", 3, 4, 1)
	require.NoError(t, err)
	world.Components.Switch.Get(lantern).On = true

	require.NoError(t, (&WhileOnChargeSystem{}).Update(world))
	assert.Equal(t, 1499, world.Components.Charge.Get(lantern).Current, "床でもオンなら照らして減る")
}
