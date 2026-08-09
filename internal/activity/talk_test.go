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

func TestTalkBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効な会話対象の場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		npc, err := lifecycle.SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "merchant")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			Params:       &gc.TalkParams{Target: npc},
		}

		ta := &TalkBehavior{}
		err = ta.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
		}

		ta := &TalkBehavior{}
		err = ta.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "talk target is not set")
	})

	t.Run("Dialogコンポーネントがない場合は不変条件違反", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// Dialogなしのエンティティを手動で作成
		npc := world.ECS.NewEntity()
		world.Components.FactionNeutral.Add(npc, &gc.FactionNeutral{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			Params:       &gc.TalkParams{Target: npc},
		}

		ta := &TalkBehavior{}
		err = ta.Validate(comp, player, world)
		require.Error(t, err)
		var ve *UserError
		require.NotErrorAs(t, err, &ve)
	})

	t.Run("FactionNeutralがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// FactionNeutralなしのエンティティを手動で作成
		npc := world.ECS.NewEntity()
		world.Components.Dialog.Add(npc, &gc.Dialog{MessageKey: "test"})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			Params:       &gc.TalkParams{Target: npc},
		}

		ta := &TalkBehavior{}
		err = ta.Validate(comp, player, world)
		var ve *UserError
		require.ErrorAs(t, err, &ve)
	})
}

func TestTalkBehavior_Info(t *testing.T) {
	t.Parallel()

	ta := &TalkBehavior{}
	info := ta.Info()

	assert.Equal(t, "Talk", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestTalkBehavior_Name(t *testing.T) {
	t.Parallel()

	ta := &TalkBehavior{}
	assert.Equal(t, gc.BehaviorTalk, ta.Name())
}

func TestTalkBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常に会話して完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		npc, err := lifecycle.SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "merchant")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			State:        gc.ActivityStateRunning,
			Params:       &gc.TalkParams{Target: npc},
		}

		ta := &TalkBehavior{}
		err = ta.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("Nameコンポーネントがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// Nameなしのエンティティを手動で作成
		npc := world.ECS.NewEntity()
		world.Components.Dialog.Add(npc, &gc.Dialog{MessageKey: "test"})
		world.Components.FactionNeutral.Add(npc, &gc.FactionNeutral{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			State:        gc.ActivityStateRunning,
			Params:       &gc.TalkParams{Target: npc},
		}

		ta := &TalkBehavior{}
		err = ta.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}
