package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
)

func TestFactionRelation(t *testing.T) {
	t.Parallel()

	t.Run("味方と敵は敵対", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		ally := world.Manager.NewEntity()
		ally.AddComponent(world.Components.FactionAlly, nil)
		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		assert.Equal(t, query.RelationHostile, query.FactionRelation(world, ally, enemy))
		assert.Equal(t, query.RelationHostile, query.FactionRelation(world, enemy, ally))
	})

	t.Run("味方同士は友好", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		ally1 := world.Manager.NewEntity()
		ally1.AddComponent(world.Components.FactionAlly, nil)
		ally2 := world.Manager.NewEntity()
		ally2.AddComponent(world.Components.FactionAlly, nil)
		assert.Equal(t, query.RelationFriendly, query.FactionRelation(world, ally1, ally2))
	})

	t.Run("敵同士は友好", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		enemy1 := world.Manager.NewEntity()
		enemy1.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		enemy2 := world.Manager.NewEntity()
		enemy2.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		assert.Equal(t, query.RelationFriendly, query.FactionRelation(world, enemy1, enemy2))
	})

	t.Run("中立と敵は中立", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		neutral := world.Manager.NewEntity()
		neutral.AddComponent(world.Components.FactionNeutral, &gc.FactionNeutral)
		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		assert.Equal(t, query.RelationNeutral, query.FactionRelation(world, neutral, enemy))
	})

	t.Run("中立と味方は中立", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		neutral := world.Manager.NewEntity()
		neutral.AddComponent(world.Components.FactionNeutral, &gc.FactionNeutral)
		ally := world.Manager.NewEntity()
		ally.AddComponent(world.Components.FactionAlly, nil)
		assert.Equal(t, query.RelationNeutral, query.FactionRelation(world, neutral, ally))
	})
}
