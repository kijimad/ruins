package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickupActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("同じタイルにアイテムがある場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		_, err = worldhelper.SpawnFieldItem(world, "木刀", 10, 10)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("同じタイルにアイテムがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// アイテムは別のタイルにある
		_, err = worldhelper.SpawnFieldItem(world, "木刀", 20, 20)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "拾えるアイテムがありません")
	})

	t.Run("位置情報がない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 位置情報なしのプレイヤーを手動で作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
		}

		pa := &PickupActivity{}
		err := pa.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "位置情報が見つかりません")
	})
}

func TestPickupActivity_Info(t *testing.T) {
	t.Parallel()

	pa := &PickupActivity{}
	info := pa.Info()

	assert.Equal(t, "拾得", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestPickupActivity_Name(t *testing.T) {
	t.Parallel()

	pa := &PickupActivity{}
	assert.Equal(t, gc.BehaviorPickup, pa.Name())
}

func TestPickupActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常にアイテムを拾って完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		item, err := worldhelper.SpawnFieldItem(world, "木刀", 10, 10)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
		}

		pa := &PickupActivity{}
		err = pa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// アイテムがバックパックに移動していることを確認
		assert.True(t, item.HasComponent(world.Components.ItemLocationInPlayerBackpack))
		// フィールドから消えていることを確認
		assert.False(t, item.HasComponent(world.Components.GridElement))
	})

	t.Run("拾えるアイテムがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "セレスティン")
		require.NoError(t, err)

		// 別のタイルにアイテムがある
		_, err = worldhelper.SpawnFieldItem(world, "木刀", 20, 20)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
		}

		pa := &PickupActivity{}
		err = pa.DoTurn(comp, player, world)

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
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
		}

		pa := &PickupActivity{}
		err := pa.DoTurn(comp, player, world)

		assert.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}
