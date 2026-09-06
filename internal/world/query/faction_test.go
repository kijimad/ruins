package query_test

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

func TestIsAreaSafe(t *testing.T) {
	t.Parallel()

	t.Run("敵がいない場合は安全", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		assert.True(t, query.IsAreaSafe(world, player))
	})

	t.Run("近くに敵がいる場合は危険", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		enemy := world.ECS.NewEntity()
		world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemy{})
		world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})

		assert.False(t, query.IsAreaSafe(world, player))
	})

	t.Run("遠くに敵がいる場合は安全", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		enemy := world.ECS.NewEntity()
		world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemy{})
		world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 15, Y: 15}})

		assert.True(t, query.IsAreaSafe(world, player))
	})

	t.Run("GridElementがない場合は危険と判定", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		assert.False(t, query.IsAreaSafe(world, player))
	})
}

func TestIsNeutral_中立派閥コンポーネントを持つエンティティはtrue(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	neutral := world.ECS.NewEntity()
	world.Components.FactionNeutral.Add(neutral, &gc.FactionNeutral{})
	assert.True(t, query.IsNeutral(world, neutral))
}

func TestIsNeutral_中立派閥コンポーネントを持たないエンティティはfalse(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	ally := world.ECS.NewEntity()
	world.Components.FactionAlly.Add(ally, &gc.FactionAlly{})
	assert.False(t, query.IsNeutral(world, ally))
}

func TestFactionRelation(t *testing.T) {
	t.Parallel()

	t.Run("味方と敵は敵対", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		ally := world.ECS.NewEntity()
		world.Components.FactionAlly.Add(ally, &gc.FactionAlly{})
		enemy := world.ECS.NewEntity()
		world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemy{})
		assert.Equal(t, query.RelationHostile, query.FactionRelation(world, ally, enemy))
		assert.Equal(t, query.RelationHostile, query.FactionRelation(world, enemy, ally))
	})

	t.Run("味方同士は友好", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		ally1 := world.ECS.NewEntity()
		world.Components.FactionAlly.Add(ally1, &gc.FactionAlly{})
		ally2 := world.ECS.NewEntity()
		world.Components.FactionAlly.Add(ally2, &gc.FactionAlly{})
		assert.Equal(t, query.RelationFriendly, query.FactionRelation(world, ally1, ally2))
	})

	t.Run("敵同士は友好", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		enemy1 := world.ECS.NewEntity()
		world.Components.FactionEnemy.Add(enemy1, &gc.FactionEnemy{})
		enemy2 := world.ECS.NewEntity()
		world.Components.FactionEnemy.Add(enemy2, &gc.FactionEnemy{})
		assert.Equal(t, query.RelationFriendly, query.FactionRelation(world, enemy1, enemy2))
	})

	t.Run("中立と敵は中立", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		neutral := world.ECS.NewEntity()
		world.Components.FactionNeutral.Add(neutral, &gc.FactionNeutral{})
		enemy := world.ECS.NewEntity()
		world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemy{})
		assert.Equal(t, query.RelationNeutral, query.FactionRelation(world, neutral, enemy))
	})

	t.Run("中立と味方は中立", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		neutral := world.ECS.NewEntity()
		world.Components.FactionNeutral.Add(neutral, &gc.FactionNeutral{})
		ally := world.ECS.NewEntity()
		world.Components.FactionAlly.Add(ally, &gc.FactionAlly{})
		assert.Equal(t, query.RelationNeutral, query.FactionRelation(world, neutral, ally))
	})
}
