package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanPlayerAct(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーターンかつAP>=0なら行動可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		// TurnBasedコンポーネントを追加する
		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
			AP: gc.IntPool{Max: 100, Current: 100},
		})

		state := query.GetTurnState(world)
		require.NotNil(t, state)
		state.Phase = gc.TurnPhasePlayer

		assert.True(t, query.CanPlayerAct(world))
	})

	t.Run("AIフェーズでは行動不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
			AP: gc.IntPool{Max: 100, Current: 100},
		})

		state := query.GetTurnState(world)
		require.NotNil(t, state)
		state.Phase = gc.TurnPhaseAI

		assert.False(t, query.CanPlayerAct(world))
	})

	t.Run("APが負の場合は行動不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
			AP: gc.IntPool{Max: 100, Current: -1},
		})

		state := query.GetTurnState(world)
		require.NotNil(t, state)
		state.Phase = gc.TurnPhasePlayer

		assert.False(t, query.CanPlayerAct(world))
	})

	t.Run("プレイヤーエンティティが存在しない場合は行動不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		assert.False(t, query.CanPlayerAct(world))
	})
}

func TestConsumeActionPoints(t *testing.T) {
	t.Parallel()

	t.Run("APを消費できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
			AP: gc.IntPool{Max: 100, Current: 100},
		})

		ok := query.ConsumeActionPoints(world, player, 30)
		assert.True(t, ok)

		tb := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		assert.Equal(t, 70, tb.AP.Current)
	})

	t.Run("TurnBasedがないエンティティではfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// TurnBasedを持たないエンティティを直接作成する
		entity := world.Manager.NewEntity().AddComponent(world.Components.Name, &gc.Name{Name: "dummy"})

		ok := query.ConsumeActionPoints(world, entity, 10)
		assert.False(t, ok)
	})
}

func TestRestoreAllActionPoints(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	tb := world.Components.TurnBased.Get(player).(*gc.TurnBased)
	initialMax := tb.AP.Max
	tb.AP.Current = 0

	err = query.RestoreAllActionPoints(world)
	require.NoError(t, err)

	tb = world.Components.TurnBased.Get(player).(*gc.TurnBased)
	assert.Greater(t, tb.AP.Current, 0, "APが回復している")
	assert.LessOrEqual(t, tb.AP.Current, initialMax, "APは最大値を超えない")
}
