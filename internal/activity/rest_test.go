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

func TestRestBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("安全な場所で有効なdurationの場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			Progress:     gc.IntPool{Max: 10},
		}

		ra := &RestBehavior{}
		err = ra.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("敵が近くにいる場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 敵を手動で作成
		enemy := world.ECS.NewEntity()
		world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemy{})
		world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			Progress:     gc.IntPool{Max: 10},
		}

		ra := &RestBehavior{}
		err = ra.Validate(comp, player, world)
		var ve *UserError
		assert.ErrorAs(t, err, &ve)
	})

	t.Run("Requiredが0以下の場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			Progress:     gc.IntPool{Max: 0},
		}

		ra := &RestBehavior{}
		err = ra.Validate(comp, player, world)
		require.ErrorIs(t, err, ErrRestInvalidDuration)
	})
}

func TestRestBehavior_performHealing(t *testing.T) {
	t.Parallel()

	t.Run("HPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// HPを減らす
		hp := world.Components.HP.Get(player)
		beforeHP := hp.Current
		hp.Current = hp.Max / 2

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			Progress:     gc.IntPool{Max: 10},
		}

		ra := &RestBehavior{}
		err = ra.performHealing(comp, player, world)
		require.NoError(t, err)

		// HPが増加したことを確認
		assert.Greater(t, hp.Current, beforeHP/2)
	})

	t.Run("HPが最大値を超えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// HPを最大値付近に設定
		hp := world.Components.HP.Get(player)
		hp.Current = hp.Max - 1

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			Progress:     gc.IntPool{Max: 10},
		}

		ra := &RestBehavior{}
		err = ra.performHealing(comp, player, world)
		require.NoError(t, err)

		// HPが最大値を超えていないことを確認
		assert.LessOrEqual(t, hp.Current, hp.Max)
	})

	t.Run("HP満タンの場合は早期完了", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// SpawnPlayerは満タンHPで作成される
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			Progress:     gc.IntPool{Max: 10},
		}

		ra := &RestBehavior{}
		err = ra.performHealing(comp, player, world)
		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("Poolsがない場合はスキップ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Poolsなしのプレイヤーを手動で作成
		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			Progress:     gc.IntPool{Max: 10},
		}

		ra := &RestBehavior{}
		err := ra.performHealing(comp, player, world)
		assert.NoError(t, err)
	})
}

// TestRestBehavior_所要はAP量で伸縮する は、同じ必要量でも毎ターン注げるAPが多いほど
// 少ないターンで完了することを固定する。着手時にターン数を凍結せず、毎ターンのAPを
// 累積するモデルであることを保証する。
func TestRestBehavior_所要はAP量で伸縮する(t *testing.T) {
	t.Parallel()

	// 指定したAP最大値で、必要量に達するまでのターン数を数える
	stepsToComplete := func(apMax int) int {
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		world.Components.TurnBased.Get(player).AP.Max = apMax
		// HP回復での早期完了を避けるため、回復が追いつかない大きな余地を作る
		hp := world.Components.HP.Get(player)
		hp.Max = 1000000
		hp.Current = 1

		rb := &RestBehavior{}
		// 休息の必要総量は NewRestActivity が Info の TotalRequiredAP から据える
		comp := NewRestActivity()
		steps := 0
		for !IsCompleted(comp) {
			require.NoError(t, rb.DoTurn(comp, player, world))
			steps++
			require.Less(t, steps, 100000, "無限ループ保険")
		}
		return steps
	}

	fast := stepsToComplete(500)
	slow := stepsToComplete(100)
	assert.Less(t, fast, slow, "APが多いほど少ないターンで休息が完了する")
}

func TestRestBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("安全な場所で1ターン進行する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// HPを減らす
		hp := world.Components.HP.Get(player)
		hp.Current = hp.Max / 2

		// 必要量を十分大きくして、1ターンでは完了しないようにする
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			Progress:     gc.IntPool{Max: 1000000},
		}

		ra := &RestBehavior{}
		err = ra.DoTurn(comp, player, world)

		require.NoError(t, err)
		// 今ターンのAP分だけ累積が進む
		assert.Equal(t, perTurnAP(player, world), comp.Progress.Current)
		assert.False(t, IsCompleted(comp))
	})

	t.Run("敵が近くにいる場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 敵を手動で作成
		enemy := world.ECS.NewEntity()
		world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemy{})
		world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			Progress:     gc.IntPool{Max: 5},
		}

		ra := &RestBehavior{}
		err = ra.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("必要量に達したら完了", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// HPを減らして満タン早期完了を避け、必要量到達での完了を確かめる
		hp := world.Components.HP.Get(player)
		hp.Current = hp.Max / 2

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			Progress:     gc.IntPool{Max: 1},
		}

		ra := &RestBehavior{}
		err = ra.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("最後のターンで完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// HPを減らす
		hp := world.Components.HP.Get(player)
		hp.Current = hp.Max / 2

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			Progress:     gc.IntPool{Max: 1},
		}

		ra := &RestBehavior{}
		err = ra.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})
}

func TestRestBehavior_Canceled(t *testing.T) {
	t.Parallel()

	t.Run("キャンセル時にログが出力される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateCanceled,
			// 実コードと同じ英語の CancelReason を使う。表示時に query.T で日本語へ訳される
			CancelReason: "rest interrupted because enemies are nearby",
		}

		ra := &RestBehavior{}
		err = ra.Canceled(comp, player, world)
		require.NoError(t, err)

		store := query.GetGameLog(world)
		recent := store.GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "Rest interrupted")
		assert.Contains(t, recent[0], "rest interrupted because enemies are nearby")
	})
}

func TestRestBehavior_Finish(t *testing.T) {
	t.Parallel()

	t.Run("完了時にログが出力される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// HPを減らす
		hp := world.Components.HP.Get(player)
		hp.Current = hp.Max / 2

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
		}

		ra := &RestBehavior{}
		err = ra.Finish(comp, player, world)
		require.NoError(t, err)

		store := query.GetGameLog(world)
		recent := store.GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "rest")
	})
}
