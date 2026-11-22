package gameaction

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

		ApplyConditionDamage(world, entity, 20, gc.CauseHypothermia)

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

		ApplyConditionDamage(world, entity, 20, gc.CauseHypothermia)

		assert.Equal(t, 0, world.Components.HP.Get(entity).Current)
		assert.True(t, world.Components.Dead.Has(entity))
	})

	t.Run("倒れた者は削らず数字も浮かべない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)
		world.Components.HP.Get(entity).Current = 10
		countEffects := func() int {
			n := 0
			q := ecs.NewFilter1[gc.VisualEffects](world.ECS).Query()
			for q.Next() {
				n++
			}
			return n
		}

		ApplyConditionDamage(world, entity, 20, gc.CauseIllness)
		require.True(t, world.Components.Dead.Has(entity))
		effects := countEffects()

		ApplyConditionDamage(world, entity, 20, gc.CauseIllness)
		assert.Equal(t, 0, world.Components.HP.Get(entity).Current)
		assert.Equal(t, effects, countEffects())
	})

	t.Run("プレイヤーが倒れると死因を記録する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		world.Components.HP.Get(player).Current = 3

		ApplyConditionDamage(world, player, 10, gc.CauseHypothermia)

		assert.True(t, world.Components.Dead.Has(player))
		rs := query.GetRunStats(world)
		require.NotNil(t, rs)
		assert.Equal(t, gc.CauseHypothermia, rs.Cause)
	})

	t.Run("敵が倒れても死因は記録しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "bat")
		require.NoError(t, err)
		world.Components.HP.Get(entity).Current = 1

		ApplyConditionDamage(world, entity, 10, gc.CauseHypothermia)

		require.True(t, world.Components.Dead.Has(entity))
		rs := query.GetRunStats(world)
		require.NotNil(t, rs)
		assert.Empty(t, rs.Cause, "敵の死では死因を記録しない")
	})
}

func TestApplyDamage_HPが尽きた死因はinjuryを記録する(t *testing.T) {
	t.Parallel()

	t.Run("敵の攻撃で倒れたプレイヤーは injury", func(t *testing.T) {
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
		assert.Equal(t, gc.CauseInjury, rs.Cause)
	})

	t.Run("自傷で倒れたプレイヤーも injury", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		world.Components.HP.Get(player).Current = 3

		ApplyDamage(world, player, 10, player)

		require.True(t, world.Components.Dead.Has(player))
		rs := query.GetRunStats(world)
		require.NotNil(t, rs)
		assert.Equal(t, gc.CauseInjury, rs.Cause)
	})

	t.Run("敵が倒れても死因は記録しない", func(t *testing.T) {
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
