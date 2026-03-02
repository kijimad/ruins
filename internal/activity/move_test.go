package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効な移動先の場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			Destination:  &gc.GridElement{X: 11, Y: 10},
		}

		ma := &MoveActivity{}
		err = ma.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("移動先がnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			Destination:  nil,
		}

		ma := &MoveActivity{}
		err = ma.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Equal(t, ErrMoveTargetNotSet, err)
	})

	t.Run("位置情報がない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 位置情報なしのプレイヤーを手動で作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			Destination:  &gc.GridElement{X: 11, Y: 10},
		}

		ma := &MoveActivity{}
		err := ma.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Equal(t, ErrMoveNoGridElement, err)
	})
}

func TestMoveActivity_Info(t *testing.T) {
	t.Parallel()

	ma := &MoveActivity{}
	info := ma.Info()

	assert.Equal(t, "移動", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestMoveActivity_Name(t *testing.T) {
	t.Parallel()

	ma := &MoveActivity{}
	assert.Equal(t, gc.BehaviorMove, ma.Name())
}

func TestMoveActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常に移動して完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			State:        gc.ActivityStateRunning,
			Destination:  &gc.GridElement{X: 11, Y: 10},
		}

		ma := &MoveActivity{}
		err = ma.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// 移動していることを確認
		gridElement := world.Components.GridElement.Get(player).(*gc.GridElement)
		assert.Equal(t, 11, int(gridElement.X))
		assert.Equal(t, 10, int(gridElement.Y))
	})

	t.Run("移動先がnilの場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			State:        gc.ActivityStateRunning,
			Destination:  nil,
		}

		ma := &MoveActivity{}
		err = ma.DoTurn(comp, player, world)

		assert.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("位置情報がない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 位置情報なしのプレイヤーを手動で作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMove,
			State:        gc.ActivityStateRunning,
			Destination:  &gc.GridElement{X: 11, Y: 10},
		}

		ma := &MoveActivity{}
		err := ma.DoTurn(comp, player, world)

		assert.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}
