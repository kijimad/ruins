package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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
	comp := NewSearchActivity(crate, world)
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

func TestSearchBehavior_Validate_探し終えた収納は探せない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)
	world.Components.Unsearched.Remove(crate)

	var ue *UserError
	require.ErrorAs(t, (&SearchBehavior{}).Validate(NewSearchActivity(crate, world), player, world), &ue)
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

func TestSearchBehavior_Validate_敵が近いと探せない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "fireball")
	require.NoError(t, err)

	var ue *UserError
	require.ErrorAs(t, (&SearchBehavior{}).Validate(NewSearchActivity(crate, world), player, world), &ue)
	assert.Equal(t, "cannot search because enemies are nearby", ue.Msg)
}

func TestSearchBehavior_DoTurn_敵が近づくと中断し収納はまだ探していないまま残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)

	b := &SearchBehavior{}
	comp := NewSearchActivity(crate, world)
	require.NoError(t, b.Validate(comp, player, world))
	require.NoError(t, b.Start(comp, player, world))
	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "fireball")
	require.NoError(t, err)
	require.NoError(t, b.DoTurn(comp, player, world))

	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, i18n.Msgid("searching interrupted because enemies are nearby"), comp.CancelReason)
	assert.True(t, world.Components.Unsearched.Has(crate), "中断すると探し直しになる")
}

func TestSearchBehavior_DoTurn_収納が消えると中断する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
	require.NoError(t, err)

	b := &SearchBehavior{}
	comp := NewSearchActivity(crate, world)
	world.ECS.RemoveEntity(crate)
	require.NoError(t, b.DoTurn(comp, player, world))

	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, i18n.Msgid("interrupted because the container is gone"), comp.CancelReason)
}

func TestSearchBehavior_Canceled_収納が残れば名前を出し消えていれば出さない(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		remove bool
		want   string
	}{
		{"残る", false, "interrupted searching Wooden Crate"},
		{"消えた", true, "interrupted searching"},
	} {
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		crate, err := lifecycle.SpawnProp(world, "wooden_crate", 10, 9)
		require.NoError(t, err)
		comp := NewSearchActivity(crate, world)
		if tc.remove {
			world.ECS.RemoveEntity(crate)
		}

		require.NoError(t, (&SearchBehavior{}).Canceled(comp, player, world))
		assert.Equal(t, []string{tc.want}, query.GetGameLog(world).GetRecent(1), tc.name)
	}
}
