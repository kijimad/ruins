package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効なdurationの場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			TurnsTotal:   1,
		}

		wa := &WaitActivity{}
		err = wa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("TurnsTotalが0以下の場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			TurnsTotal:   0,
		}

		wa := &WaitActivity{}
		err = wa.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "待機時間が無効")
	})
}

func TestWaitActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("1ターン進行する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   5,
			TurnsLeft:    3,
		}

		wa := &WaitActivity{}
		err = wa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, 2, comp.TurnsLeft)
	})

	t.Run("TurnsLeftが0以下なら完了", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   5,
			TurnsLeft:    0,
		}

		wa := &WaitActivity{}
		err = wa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("最後のターンで完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   5,
			TurnsLeft:    1,
		}

		wa := &WaitActivity{}
		err = wa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Equal(t, 0, comp.TurnsLeft)
	})
}

func TestWaitActivity_Finish(t *testing.T) {
	t.Parallel()

	t.Run("1ターン待機ではログを出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			TurnsTotal:   1,
		}

		wa := &WaitActivity{}
		err = wa.Finish(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("複数ターン待機ではログを出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			TurnsTotal:   5,
		}

		wa := &WaitActivity{}
		err = wa.Finish(comp, player, world)
		assert.NoError(t, err)
	})
}
