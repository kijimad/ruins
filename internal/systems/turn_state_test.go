package systems_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTurnState(t *testing.T) {
	t.Parallel()

	t.Run("シングルトンからターン状態を取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		state, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)
		require.NotNil(t, state, "TurnStateシングルトンが存在する")
		assert.Equal(t, gc.TurnPhasePlayer, state.Phase, "初期フェーズはPlayerTurn")
		assert.Equal(t, 1, state.TurnNumber, "初期ターン番号は1")
	})
}

func TestTurnStateDirectManipulation(t *testing.T) {
	t.Parallel()

	t.Run("Phaseを直接変更できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		state, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)

		// PlayerTurn -> AITurn
		state.Phase = gc.TurnPhaseAI
		assert.Equal(t, gc.TurnPhaseAI, state.Phase)

		// AITurn -> TurnEnd
		state.Phase = gc.TurnPhaseEnd
		assert.Equal(t, gc.TurnPhaseEnd, state.Phase)

		// TurnEnd -> PlayerTurn
		state.Phase = gc.TurnPhasePlayer
		assert.Equal(t, gc.TurnPhasePlayer, state.Phase)
	})

	t.Run("TurnNumberを直接変更できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		state, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)

		assert.Equal(t, 1, state.TurnNumber)

		state.TurnNumber++
		assert.Equal(t, 2, state.TurnNumber)

		state.TurnNumber++
		assert.Equal(t, 3, state.TurnNumber)
	})
}

func TestTurnCycle(t *testing.T) {
	t.Parallel()

	t.Run("完全なターンサイクル", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		state, err := worldhelper.GetTurnState(world)
		require.NoError(t, err)

		// 初期状態
		assert.Equal(t, gc.TurnPhasePlayer, state.Phase)
		assert.Equal(t, 1, state.TurnNumber)

		// PlayerTurn → AITurn
		state.Phase = gc.TurnPhaseAI
		assert.Equal(t, gc.TurnPhaseAI, state.Phase)

		// AITurn → TurnEnd
		state.Phase = gc.TurnPhaseEnd
		assert.Equal(t, gc.TurnPhaseEnd, state.Phase)

		// TurnEnd → PlayerTurn (新ターン)
		state.TurnNumber++
		state.Phase = gc.TurnPhasePlayer
		assert.Equal(t, gc.TurnPhasePlayer, state.Phase)
		assert.Equal(t, 2, state.TurnNumber)
	})
}
