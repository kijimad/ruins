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

func TestDropActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効なドロップ対象の場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}},
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       nil,
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ドロップ対象が指定されていません")
	})

	t.Run("バックパック内にないアイテムの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		// バックパック外のアイテムを手動で作成
		item := world.ECS.NewEntity()
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}},
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "バックパック内にありません")
	})

	t.Run("Destinationがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "目的地が指定されていません")
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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}},
		}

		da := &DropActivity{}
		err = da.performDropActivity(comp, player, world)
		require.NoError(t, err)

		// アイテムがフィールドに配置されていることを確認
		assert.True(t, world.Components.GridElement.Has(item))
		gridElement := world.Components.GridElement.Get(item)
		assert.Equal(t, 10, int(gridElement.X))
		assert.Equal(t, 10, int(gridElement.Y))

		// バックパックから削除されていることを確認
		assert.True(t, world.Components.LocationOnField.Has(item))

		// ドロップログが出力されていることを確認する
		store := query.GetGameLog(world)
		recent := store.GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "を置いた")
	})

	t.Run("Destinationがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
		}

		da := &DropActivity{}
		err = da.performDropActivity(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "目的地が指定されていません")
	})
}

func TestDropActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常にドロップして完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Target:       &item,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}},
		}

		da := &DropActivity{}
		err = da.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("Destinationがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Target:       &item,
		}

		da := &DropActivity{}
		err = da.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestDropActivity_performDropActivity_AdjacentTile(t *testing.T) {
	t.Parallel()

	t.Run("隣接タイルにアイテムをドロップできる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		// プレイヤーの右隣にドロップ
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}},
		}

		da := &DropActivity{}
		err = da.performDropActivity(comp, player, world)
		require.NoError(t, err)

		assert.True(t, world.Components.GridElement.Has(item))
		gridElement := world.Components.GridElement.Get(item)
		assert.Equal(t, 11, int(gridElement.X))
		assert.Equal(t, 10, int(gridElement.Y))
		assert.True(t, world.Components.LocationOnField.Has(item))
	})

	t.Run("斜め隣接タイルにアイテムをドロップできる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		// 右下斜めにドロップ
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 11}},
		}

		da := &DropActivity{}
		err = da.performDropActivity(comp, player, world)
		require.NoError(t, err)

		gridElement := world.Components.GridElement.Get(item)
		assert.Equal(t, 11, int(gridElement.X))
		assert.Equal(t, 11, int(gridElement.Y))
	})
}

func TestDropActivity_FixtureDerivedItem(t *testing.T) {
	t.Parallel()

	t.Run("固定物由来アイテムをドロップすると Fixed コンポーネントが保持される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		// 固定物を拾った状態をシミュレート: Fixed+Item+BlockPassがバックパックにある
		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.BlockPass.Add(prop, &gc.BlockPass{})
		require.NoError(t, lifecycle.MoveToBackpack(world, prop, player))

		// ドロップ実行
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &prop,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}},
		}

		da := &DropActivity{}
		err = da.performDropActivity(comp, player, world)
		require.NoError(t, err)

		// Fixed コンポーネントが保持されていることを確認
		assert.True(t, world.Components.Fixed.Has(prop))
		// BlockPassも保持されていることを確認
		assert.True(t, world.Components.BlockPass.Has(prop))
		// フィールドに配置されていることを確認
		assert.True(t, world.Components.LocationOnField.Has(prop))
		assert.True(t, world.Components.GridElement.Has(prop))
		gridElement := world.Components.GridElement.Get(prop)
		assert.Equal(t, 11, int(gridElement.X))
		assert.Equal(t, 10, int(gridElement.Y))
	})
}

func TestPickupAndDropRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("通常アイテムの拾得とドロップの往復", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		// 拾う
		pickupComp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}},
		}

		pa := &PickupActivity{}
		err = pa.DoTurn(pickupComp, player, world)
		require.NoError(t, err)

		assert.True(t, world.Components.LocationInBackpack.Has(item))
		assert.False(t, world.Components.GridElement.Has(item))

		// ドロップ
		dropComp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Target:       &item,
			Destination:  &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 9}},
		}

		da := &DropActivity{}
		err = da.DoTurn(dropComp, player, world)
		require.NoError(t, err)

		assert.True(t, world.Components.LocationOnField.Has(item))
		gridElement := world.Components.GridElement.Get(item)
		assert.Equal(t, 9, int(gridElement.X))
		assert.Equal(t, 9, int(gridElement.Y))
		// 通常アイテムは Fixed コンポーネントを持たない
		assert.False(t, world.Components.Fixed.Has(item))
	})
}
