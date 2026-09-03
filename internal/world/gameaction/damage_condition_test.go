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

func TestApplyConditionDamage(t *testing.T) {
	t.Parallel()

	t.Run("HPを削る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)
		hp := world.Components.HP.Get(entity)
		hp.Max = 100
		hp.Current = 50

		ApplyConditionDamage(world, entity, 20, gc.CauseFrozen)

		assert.Equal(t, 30, world.Components.HP.Get(entity).Current)
		assert.False(t, world.Components.Dead.Has(entity))
	})

	t.Run("致死ダメージでDeadが付く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)
		hp := world.Components.HP.Get(entity)
		hp.Max = 100
		hp.Current = 10

		ApplyConditionDamage(world, entity, 20, gc.CauseFrozen)

		assert.Equal(t, 0, world.Components.HP.Get(entity).Current)
		assert.True(t, world.Components.Dead.Has(entity))
	})

	t.Run("プレイヤーが倒れると死因を記録する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		world.Components.HP.Get(player).Current = 3

		ApplyConditionDamage(world, player, 10, gc.CauseFrozen)

		assert.True(t, world.Components.Dead.Has(player))
		rs := query.GetRunStats(world)
		require.NotNil(t, rs)
		assert.Equal(t, gc.CauseFrozen, rs.Cause)
	})

	t.Run("敵が倒れても死因は記録しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)
		world.Components.HP.Get(entity).Current = 1

		ApplyConditionDamage(world, entity, 10, gc.CauseFrozen)

		require.True(t, world.Components.Dead.Has(entity))
		rs := query.GetRunStats(world)
		require.NotNil(t, rs)
		assert.Empty(t, rs.Cause, "敵の死では死因を記録しない")
	})
}

func TestApplyDamage_戦闘死はkilledを記録する(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーの戦闘死は killed", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "bat")
		require.NoError(t, err)
		world.Components.HP.Get(player).Current = 3

		ApplyDamage(world, player, 10, enemy)

		require.True(t, world.Components.Dead.Has(player))
		rs := query.GetRunStats(world)
		require.NotNil(t, rs)
		assert.Equal(t, gc.CauseKilled, rs.Cause)
	})

	t.Run("敵の戦闘死では死因を記録しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)
		world.Components.HP.Get(enemy).Current = 1

		ApplyDamage(world, enemy, 10, player)

		require.True(t, world.Components.Dead.Has(enemy))
		rs := query.GetRunStats(world)
		require.NotNil(t, rs)
		assert.Empty(t, rs.Cause)
	})
}
