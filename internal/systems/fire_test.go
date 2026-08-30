package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFireSystem_Update(t *testing.T) {
	t.Parallel()

	t.Run("残量を毎ターン減らす", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 3})

		require.NoError(t, sys.Update(world))
		assert.Equal(t, 2, world.Components.Burning.Get(fire).Remaining)

		require.NoError(t, sys.Update(world))
		assert.Equal(t, 1, world.Components.Burning.Get(fire).Remaining)
	})

	t.Run("残量が尽きると鎮火して火が消える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 1})

		// 残量1が0になり Burning が外れ、火のエンティティは Dead になって除去される
		require.NoError(t, sys.Update(world))
		assert.False(t, world.Components.Burning.Has(fire), "残量が尽きれば鎮火する")
		assert.True(t, world.Components.Dead.Has(fire), "燃え尽きた火は Dead になり dead_cleanup が消す")
	})

	t.Run("残量ぶんだけ燃えて最後に鎮火する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &FireSystem{}

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 3})

		for range 2 {
			require.NoError(t, sys.Update(world))
			assert.True(t, world.Components.Burning.Has(fire))
		}
		// 3ターン目で残量0になり鎮火する
		require.NoError(t, sys.Update(world))
		assert.False(t, world.Components.Burning.Has(fire))
	})
}
