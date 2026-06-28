package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("隊員がバックパック内のアイテムを転送できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		// 隊員のバックパックにアイテムを入れる
		item, err := lifecycle.SpawnFieldItem(world, "木刀", 5, 5, 1)
		require.NoError(t, err)
		err = lifecycle.MoveToBackpack(world, item, member)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			Target:       &item,
		}

		ta := &TransferActivity{}
		err = ta.Validate(comp, member, world)
		assert.NoError(t, err)
	})

	t.Run("Targetが指定されていない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
		}

		ta := &TransferActivity{}
		err = ta.Validate(comp, member, world)
		assert.Error(t, err)
	})

	t.Run("隊員でないエンティティはエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)
		err = lifecycle.MoveToBackpack(world, item, player)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			Target:       &item,
		}

		ta := &TransferActivity{}
		err = ta.Validate(comp, player, world)
		assert.Error(t, err)
	})
}

func TestTransferActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("アイテムがリーダーのバックパックに移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		// 隊員のバックパックにアイテムを入れる
		item, err := lifecycle.SpawnFieldItem(world, "木刀", 5, 5, 1)
		require.NoError(t, err)
		err = lifecycle.MoveToBackpack(world, item, member)
		require.NoError(t, err)

		// 隊員のバックパックにあることを確認
		loc := world.Components.LocationInBackpack.Get(item).(*gc.LocationInBackpack)
		assert.Equal(t, member, loc.Owner)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorTransfer,
			State:        gc.ActivityStateRunning,
			Target:       &item,
			TurnsTotal:   1,
			TurnsLeft:    1,
		}

		ta := &TransferActivity{}
		err = ta.DoTurn(comp, member, world)
		require.NoError(t, err)

		// リーダーのバックパックに移動したことを確認
		loc = world.Components.LocationInBackpack.Get(item).(*gc.LocationInBackpack)
		assert.Equal(t, leader, loc.Owner)
	})
}

func TestTransferActivity_Info(t *testing.T) {
	t.Parallel()
	ta := &TransferActivity{}
	info := ta.Info()
	assert.Equal(t, "転送", info.Name)
	assert.Equal(t, gc.BehaviorTransfer, ta.Name())
}

func testAbilities() gc.Abilities {
	return gc.Abilities{
		Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
		Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
	}
}
