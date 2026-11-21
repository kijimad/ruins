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

func TestLightChargeSystem_床で点いた松明は燃え尽きて消える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	torch, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	world.Components.LightSource.Get(torch).Enabled = true
	world.Components.Charge.Get(torch).Current = 2
	sys := &LightChargeSystem{}

	require.NoError(t, sys.Update(world))
	assert.Equal(t, 1, world.Components.Charge.Get(torch).Current, "照らしている間は1ターンに1減る")

	query.GetVisionState(world).ConsumePendingUpdate()
	logs := query.GetGameLog(world).Count()
	require.NoError(t, sys.Update(world))
	assert.False(t, world.ECS.Alive(torch), "尽きると消える")
	assert.Equal(t, logs, query.GetGameLog(world).Count(), "床の灯りが尽きてもログを出さない")
	assert.True(t, query.GetVisionState(world).ConsumePendingUpdate(), "消えると視界を計算し直させる")
}

func TestLightChargeSystem_照らしていない灯りは減らない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	unlit, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	carried, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	world.Components.LightSource.Get(carried).Enabled = true
	require.NoError(t, lifecycle.MoveToBackpack(world, carried, player))

	require.NoError(t, (&LightChargeSystem{}).Update(world))
	assert.Equal(t, 600, world.Components.Charge.Get(unlit).Current, "消えた床の松明は減らない")
	assert.Equal(t, 600, world.Components.Charge.Get(carried).Current, "背負った松明は減らない")
}

func TestLightChargeSystem_手持ち欄の灯りは2本とも燃え尽きる(t *testing.T) {
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

	require.NoError(t, (&LightChargeSystem{}).Update(world))
	for _, torch := range torches {
		assert.False(t, world.ECS.Alive(torch), "同じターンに2本とも燃え尽きる")
	}
	assert.True(t, world.Components.StatsChanged.Has(player), "装備の灯りを写し直させる")
	entries := query.GetGameLog(world).GetRecentEntries(1)
	require.Len(t, entries, 1)
	assert.Equal(t, query.T(world, "%s burned out.", "Torch"), entries[0].Text())
}
