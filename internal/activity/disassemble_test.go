package activity

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func itemCount(t *testing.T, world w.World, id string) int {
	t.Helper()
	e, ok := query.FindStackInInventory(world, id)
	if !ok {
		return 0
	}
	return query.GetEntityCount(world, e)
}

func TestDisassembleBehavior_重さ1kgごとに1ターンかけて材質の素材に戻す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	hammer, err := lifecycle.SpawnBackpackItem(world, "hammer", 1)
	require.NoError(t, err)
	before := itemCount(t, world, "scrap_iron")

	b := &DisassembleBehavior{}
	comp := NewDisassembleActivity(hammer, world)
	assert.Equal(t, 4, comp.Progress.Max)
	require.NoError(t, b.Validate(comp, player, world))
	require.NoError(t, b.Start(comp, player, world))
	for range comp.Progress.Max {
		require.NoError(t, b.DoTurn(comp, player, world))
	}
	assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	require.NoError(t, b.Finish(comp, player, world))

	assert.False(t, world.ECS.Alive(hammer))
	assert.Equal(t, before+4, itemCount(t, world, "scrap_iron"))
}

func TestDisassembleBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("戻し先の素材そのものは分解できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		scrap, err := lifecycle.SpawnBackpackItem(world, "scrap_iron", 1)
		require.NoError(t, err)

		var ue *UserError
		require.ErrorAs(t, (&DisassembleBehavior{}).Validate(NewDisassembleActivity(scrap, world), player, world), &ue)
		assert.Equal(t, query.T(world, "this item cannot be disassembled"), ue.Msg)
	})

	t.Run("所持品に無いアイテムは分解できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		onField, err := lifecycle.SpawnFieldItem(world, "hammer", 11, 10, 1)
		require.NoError(t, err)

		var ue *UserError
		require.ErrorAs(t, (&DisassembleBehavior{}).Validate(NewDisassembleActivity(onField, world), player, world), &ue)
		assert.Equal(t, query.T(world, "the item is not in your inventory"), ue.Msg)
	})
}

func TestDisassembleBehavior_DoTurn_途中でアイテムが無くなると中断する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	hammer, err := lifecycle.SpawnBackpackItem(world, "hammer", 1)
	require.NoError(t, err)

	b := &DisassembleBehavior{}
	comp := NewDisassembleActivity(hammer, world)
	require.NoError(t, b.Validate(comp, player, world))
	require.NoError(t, b.Start(comp, player, world))
	require.NoError(t, b.DoTurn(comp, player, world))
	require.NoError(t, lifecycle.ChangeItemCount(world, hammer, -1))
	require.NoError(t, b.DoTurn(comp, player, world))

	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
}

func TestDisassembleBehavior_DoTurn_途中で敵が近づくと中断する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	hammer, err := lifecycle.SpawnBackpackItem(world, "hammer", 1)
	require.NoError(t, err)

	b := &DisassembleBehavior{}
	comp := NewDisassembleActivity(hammer, world)
	require.NoError(t, b.Validate(comp, player, world))
	require.NoError(t, b.Start(comp, player, world))
	require.NoError(t, b.DoTurn(comp, player, world))
	require.Equal(t, gc.ActivityStateRunning, comp.State)

	_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 11}, "fireball")
	require.NoError(t, err)
	require.NoError(t, b.DoTurn(comp, player, world))

	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, i18n.Msgid("disassembly interrupted because enemies are nearby"), comp.CancelReason)
	assert.True(t, world.ECS.Alive(hammer))
}

func TestDisassembleBehavior_Canceled(t *testing.T) {
	t.Parallel()

	t.Run("対象が残っていれば名前を出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		hammer, err := lifecycle.SpawnBackpackItem(world, "hammer", 1)
		require.NoError(t, err)

		require.NoError(t, (&DisassembleBehavior{}).Canceled(NewDisassembleActivity(hammer, world), player, world))

		assert.Equal(t, []string{query.T(world, "interrupted disassembling %s", query.GetEntityName(hammer, world))}, query.GetGameLog(world).GetRecent(1))
	})

	t.Run("対象が消えていれば名前を出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		hammer, err := lifecycle.SpawnBackpackItem(world, "hammer", 1)
		require.NoError(t, err)
		comp := NewDisassembleActivity(hammer, world)
		require.NoError(t, lifecycle.ChangeItemCount(world, hammer, -1))

		require.NoError(t, (&DisassembleBehavior{}).Canceled(comp, player, world))

		assert.Equal(t, []string{query.T(world, "interrupted disassembling")}, query.GetGameLog(world).GetRecent(1))
	})
}
