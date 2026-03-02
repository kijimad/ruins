package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDropActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効なドロップ対象の場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := worldhelper.SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       nil,
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ドロップ対象が指定されていません")
	})

	t.Run("バックパック内にないアイテムの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// バックパック外のアイテムを手動で作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "バックパック内にありません")
	})

	t.Run("プレイヤーの位置情報がない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 位置情報なしのプレイヤーを手動で作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		item, err := worldhelper.SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "位置情報が見つかりません")
	})
}

func TestDropActivity_Info(t *testing.T) {
	t.Parallel()

	da := &DropActivity{}
	info := da.Info()

	assert.Equal(t, "ドロップ", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestDropActivity_Name(t *testing.T) {
	t.Parallel()

	da := &DropActivity{}
	assert.Equal(t, gc.BehaviorDrop, da.Name())
}

func TestDropActivity_performDropActivity(t *testing.T) {
	t.Parallel()

	t.Run("アイテムをフィールドにドロップできる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := worldhelper.SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
		}

		da := &DropActivity{}
		err = da.performDropActivity(comp, player, world)
		require.NoError(t, err)

		// アイテムがフィールドに配置されていることを確認
		assert.True(t, item.HasComponent(world.Components.GridElement))
		gridElement := world.Components.GridElement.Get(item).(*gc.GridElement)
		assert.Equal(t, 10, int(gridElement.X))
		assert.Equal(t, 10, int(gridElement.Y))

		// バックパックから削除されていることを確認
		assert.True(t, item.HasComponent(world.Components.ItemLocationOnField))
	})

	t.Run("位置情報がない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 位置情報なしのプレイヤーを手動で作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// アイテムも手動で作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
		}

		da := &DropActivity{}
		err := da.performDropActivity(comp, player, world)
		assert.Error(t, err)
	})
}

func TestDropActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常にドロップして完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := worldhelper.SpawnItem(world, "木刀", 1, gc.ItemLocationInPlayerBackpack)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Target:       &item,
		}

		da := &DropActivity{}
		err = da.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("エラー時はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 位置情報なしのプレイヤーを手動で作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})

		// アイテムも手動で作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Target:       &item,
		}

		da := &DropActivity{}
		err := da.DoTurn(comp, player, world)

		assert.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}
