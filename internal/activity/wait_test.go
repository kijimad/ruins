package activity

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

func TestWaitBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効な待機回数の場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			Progress:     gc.IntPool{Max: 1},
		}

		wa := &WaitBehavior{}
		err = wa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Requiredが0以下の場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			Progress:     gc.IntPool{Max: 0},
		}

		wa := &WaitBehavior{}
		err = wa.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "待機回数が無効")
	})
}

func TestWaitBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("1ターン進行する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
			Progress:     gc.IntPool{Max: 5, Current: 2},
		}

		wa := &WaitBehavior{}
		err = wa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, 3, comp.Progress.Current)
		assert.False(t, IsCompleted(comp))
	})

	t.Run("必要量に達したら完了", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
			Progress:     gc.IntPool{Max: 5, Current: 4},
		}

		wa := &WaitBehavior{}
		err = wa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("最後のターンで完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
			Progress:     gc.IntPool{Max: 5, Current: 4},
		}

		wa := &WaitBehavior{}
		err = wa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Equal(t, 5, comp.Progress.Current)
	})
}

func TestWaitBehavior_Finish(t *testing.T) {
	t.Parallel()

	t.Run("1ターン待機ではログを出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			Progress:     gc.IntPool{Max: 1},
		}

		wa := &WaitBehavior{}
		err = wa.Finish(comp, player, world)
		require.NoError(t, err)

		store := query.GetGameLog(world)
		assert.Equal(t, 0, store.Count())
	})

	t.Run("複数ターン待機ではログを出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			Progress:     gc.IntPool{Max: 5},
		}

		wa := &WaitBehavior{}
		err = wa.Finish(comp, player, world)
		require.NoError(t, err)

		store := query.GetGameLog(world)
		recent := store.GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "待機を終了した")
	})
}

func TestWaitBehavior_長い待機は敵接近で中断する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := newDisassembleTestPlayer(world)
	spawnHostileAt(world, 11, 10)

	wa := &WaitBehavior{}
	comp := &gc.Activity{BehaviorName: gc.BehaviorWait, State: gc.ActivityStateRunning, Progress: gc.IntPool{Max: 5}}

	require.NoError(t, wa.DoTurn(comp, actor, world))
	assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	assert.Equal(t, "周囲に敵がいるため待機を中断", comp.CancelReason)
}

func TestWaitBehavior_1ターンの待機は敵が隣接していても完結する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := newDisassembleTestPlayer(world)
	spawnHostileAt(world, 11, 10)

	wa := &WaitBehavior{}
	comp := &gc.Activity{BehaviorName: gc.BehaviorWait, State: gc.ActivityStateRunning, Progress: gc.IntPool{Max: 1}}

	require.NoError(t, wa.DoTurn(comp, actor, world))
	require.NoError(t, wa.DoTurn(comp, actor, world))
	assert.Equal(t, gc.ActivityStateCompleted, comp.State,
		"ターン送りやAIの手番調整は中断の対象外にするべき")
}
