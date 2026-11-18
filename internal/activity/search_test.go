package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchBehavior_容量10kgごとに1ターンかけて探し終えると収納を開く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)
	require.True(t, world.Components.Unsearched.Has(crate), "前提: 生成した収納はまだ探していない")

	b := &SearchBehavior{}
	comp := NewSearchActivity(crate, false, world)
	assert.Equal(t, 2, comp.Progress.Max, "20kg の木箱は2ターン")
	require.NoError(t, b.Validate(comp, player, world))
	require.NoError(t, b.Start(comp, player, world))
	for range comp.Progress.Max {
		require.NoError(t, b.DoTurn(comp, player, world))
	}
	assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	require.NoError(t, b.Finish(comp, player, world))

	assert.False(t, world.Components.Unsearched.Has(crate))
	req := lifecycle.ConsumeStateChange(world)
	require.NotNil(t, req)
	assert.Equal(t, gc.OpenStorage{StorageEntity: crate}, req.Payload)
}

func TestSearchBehavior_拾うメニューから探し始めると拾うメニューの上に収納を開く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)

	b := &SearchBehavior{}
	comp := NewSearchActivity(crate, true, world)
	require.NoError(t, b.Validate(comp, player, world))
	require.NoError(t, b.Start(comp, player, world))
	for range comp.Progress.Max {
		require.NoError(t, b.DoTurn(comp, player, world))
	}
	require.NoError(t, b.Finish(comp, player, world))

	req := lifecycle.ConsumeStateChange(world)
	require.NotNil(t, req)
	assert.Equal(t, gc.OpenStorage{StorageEntity: crate, OverPickup: true}, req.Payload)
}

func TestSearchBehavior_Validate_探し終えた収納は探せない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)
	world.Components.Unsearched.Remove(crate)

	var ue *UserError
	require.ErrorAs(t, (&SearchBehavior{}).Validate(NewSearchActivity(crate, false, world), player, world), &ue)
}

func TestExecuteInteraction_まだ探していない収納は探すアクティビティを始める(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)

	result, err := ExecuteInteraction(player, crate, gc.InteractionStorage, world)
	require.NoError(t, err)

	assert.Equal(t, gc.BehaviorSearch, result.ActivityName)
	assert.Nil(t, lifecycle.ConsumeStateChange(world), "探し終えるまで収納画面は開かない")
	assert.True(t, world.Components.Unsearched.Has(crate))
}
