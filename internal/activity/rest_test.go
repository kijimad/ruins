package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsAreaSafe(t *testing.T) {
	t.Parallel()

	t.Run("敵がいない場合は安全", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		assert.True(t, isAreaSafe(player, world))
	})

	t.Run("近くに敵がいる場合は危険", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})

		assert.False(t, isAreaSafe(player, world))
	})

	t.Run("遠くに敵がいる場合は安全", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 15, Y: 15})

		assert.True(t, isAreaSafe(player, world))
	})

	t.Run("GridElementがない場合は危険と判定", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		assert.False(t, isAreaSafe(player, world))
	})
}

func TestRestActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("安全な場所で有効なdurationの場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			TurnsTotal:   10,
		}

		ra := &RestActivity{}
		err = ra.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("敵が近くにいる場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 敵を手動で作成
		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			TurnsTotal:   10,
		}

		ra := &RestActivity{}
		err = ra.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "敵がいる")
	})

	t.Run("TurnsTotalが0以下の場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			TurnsTotal:   0,
		}

		ra := &RestActivity{}
		err = ra.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "休息時間が無効")
	})
}

func TestRestActivity_performHealing(t *testing.T) {
	t.Parallel()

	t.Run("HPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// HPを減らす
		pools := world.Components.Pools.Get(player).(*gc.Pools)
		beforeHP := pools.HP.Current
		pools.HP.Current = pools.HP.Max / 2

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			TurnsTotal:   10,
			TurnsLeft:    5,
		}

		ra := &RestActivity{}
		err = ra.performHealing(comp, player, world)
		require.NoError(t, err)

		// HPが増加したことを確認
		assert.Greater(t, pools.HP.Current, beforeHP/2)
	})

	t.Run("HPが最大値を超えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// HPを最大値付近に設定
		pools := world.Components.Pools.Get(player).(*gc.Pools)
		pools.HP.Current = pools.HP.Max - 1

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			TurnsTotal:   10,
			TurnsLeft:    5,
		}

		ra := &RestActivity{}
		err = ra.performHealing(comp, player, world)
		require.NoError(t, err)

		// HPが最大値を超えていないことを確認
		assert.LessOrEqual(t, pools.HP.Current, pools.HP.Max)
	})

	t.Run("HP満タンの場合は早期完了", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// SpawnPlayerは満タンHPで作成される
		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   10,
			TurnsLeft:    5,
		}

		ra := &RestActivity{}
		err = ra.performHealing(comp, player, world)
		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("Poolsがない場合はスキップ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Poolsなしのプレイヤーを手動で作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			TurnsTotal:   10,
			TurnsLeft:    5,
		}

		ra := &RestActivity{}
		err := ra.performHealing(comp, player, world)
		assert.NoError(t, err)
	})
}

func TestRestActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("安全な場所で1ターン進行する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// HPを減らす
		pools := world.Components.Pools.Get(player).(*gc.Pools)
		pools.HP.Current = pools.HP.Max / 2

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   5,
			TurnsLeft:    3,
		}

		ra := &RestActivity{}
		err = ra.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, 2, comp.TurnsLeft)
	})

	t.Run("敵が近くにいる場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 敵を手動で作成
		enemy := world.Manager.NewEntity()
		enemy.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemy)
		enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   5,
			TurnsLeft:    3,
		}

		ra := &RestActivity{}
		err = ra.DoTurn(comp, player, world)

		assert.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("TurnsLeftが0以下なら完了", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   5,
			TurnsLeft:    0,
		}

		ra := &RestActivity{}
		err = ra.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("最後のターンで完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// HPを減らす
		pools := world.Components.Pools.Get(player).(*gc.Pools)
		pools.HP.Current = pools.HP.Max / 2

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   5,
			TurnsLeft:    1,
		}

		ra := &RestActivity{}
		err = ra.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Equal(t, 0, comp.TurnsLeft)
	})
}
