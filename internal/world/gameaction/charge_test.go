package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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

func TestConsumeCharge_手持ちの灯りの段階が下がるとログを出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	torch, err := lifecycle.SpawnFieldItem(world, "torch", 3, 4, 1)
	require.NoError(t, err)
	lifecycle.MoveToEquip(world, torch, player, gc.SlotHand2)
	world.Components.Charge.Get(torch).Current = 482

	ConsumeCharge(world, torch, 1)
	logs := query.GetGameLog(world).Count()
	ConsumeCharge(world, torch, 1)

	require.Equal(t, logs+1, query.GetGameLog(world).Count(), "5段から4段に下がったときだけ出す")
	entries := query.GetGameLog(world).GetRecentEntries(1)
	assert.Equal(t, query.T(world, "%s is burning down.", "Torch"), entries[0].Text())
}
