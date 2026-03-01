package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTalkActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効な会話対象の場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		npc, err := worldhelper.SpawnNeutralNPC(world, 11, 10, "商人")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			Target:       &npc,
		}

		ta := &TalkActivity{}
		err = ta.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			Target:       nil,
		}

		ta := &TalkActivity{}
		err = ta.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "会話対象が指定されていません")
	})

	t.Run("Dialogコンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// Dialogなしのエンティティを手動で作成
		npc := world.Manager.NewEntity()
		npc.AddComponent(world.Components.FactionNeutral, &gc.FactionNeutral)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			Target:       &npc,
		}

		ta := &TalkActivity{}
		err = ta.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "対象エンティティは会話できません")
	})

	t.Run("FactionNeutralがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// FactionNeutralなしのエンティティを手動で作成
		npc := world.Manager.NewEntity()
		npc.AddComponent(world.Components.Dialog, &gc.Dialog{MessageKey: "test"})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			Target:       &npc,
		}

		ta := &TalkActivity{}
		err = ta.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "対象エンティティは中立派閥ではありません")
	})
}

func TestTalkActivity_Info(t *testing.T) {
	t.Parallel()

	ta := &TalkActivity{}
	info := ta.Info()

	assert.Equal(t, "会話", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestTalkActivity_Name(t *testing.T) {
	t.Parallel()

	ta := &TalkActivity{}
	assert.Equal(t, gc.BehaviorTalk, ta.Name())
}

func TestTalkActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常に会話して完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		npc, err := worldhelper.SpawnNeutralNPC(world, 11, 10, "商人")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			State:        gc.ActivityStateRunning,
			Target:       &npc,
		}

		ta := &TalkActivity{}
		err = ta.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("Nameコンポーネントがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// Nameなしのエンティティを手動で作成
		npc := world.Manager.NewEntity()
		npc.AddComponent(world.Components.Dialog, &gc.Dialog{MessageKey: "test"})
		npc.AddComponent(world.Components.FactionNeutral, &gc.FactionNeutral)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTalk,
			State:        gc.ActivityStateRunning,
			Target:       &npc,
		}

		ta := &TalkActivity{}
		err = ta.DoTurn(comp, player, world)

		assert.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}
