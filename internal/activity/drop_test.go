package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &destination,
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// バックパック外のアイテムを手動で作成
		item := world.World.NewEntity()
		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &destination,
		}

		da := &DropActivity{}
		err = da.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "バックパック内にありません")
	})

	t.Run("Destinationがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &destination,
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Target:       &item,
			Destination:  &destination,
		}

		da := &DropActivity{}
		err = da.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("Destinationがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		// プレイヤーの右隣にドロップ
		destination := gc.GridElement{X: 11, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &destination,
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
		require.NoError(t, err)

		// 右下斜めにドロップ
		destination := gc.GridElement{X: 11, Y: 11}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &item,
			Destination:  &destination,
		}

		da := &DropActivity{}
		err = da.performDropActivity(comp, player, world)
		require.NoError(t, err)

		gridElement := world.Components.GridElement.Get(item)
		assert.Equal(t, 11, int(gridElement.X))
		assert.Equal(t, 11, int(gridElement.Y))
	})
}

func TestDropActivity_PropDerivedItem(t *testing.T) {
	t.Parallel()

	t.Run("Prop由来アイテムをドロップするとPropコンポーネントが保持される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// Propを拾った状態をシミュレート: Prop+Item+BlockPassがバックパックにある
		prop := world.World.NewEntity()
		world.Components.Prop.Add(prop, nil)
		world.Components.Name.Add(prop, &gc.Name{Name: "テストProp"})
		world.Components.BlockPass.Add(prop, &gc.BlockPass{})
		require.NoError(t, lifecycle.MoveToBackpack(world, prop, player))

		// ドロップ実行
		destination := gc.GridElement{X: 11, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Target:       &prop,
			Destination:  &destination,
		}

		da := &DropActivity{}
		err = da.performDropActivity(comp, player, world)
		require.NoError(t, err)

		// Propコンポーネントが保持されていることを確認
		assert.True(t, world.Components.Prop.Has(prop))
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

		player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		// 拾う
		pickupDest := gc.GridElement{X: 10, Y: 10}
		pickupComp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Destination:  &pickupDest,
		}

		pa := &PickupActivity{}
		err = pa.DoTurn(pickupComp, player, world)
		require.NoError(t, err)

		assert.True(t, world.Components.LocationInBackpack.Has(item))
		assert.False(t, world.Components.GridElement.Has(item))

		// ドロップ
		dropDest := gc.GridElement{X: 9, Y: 9}
		dropComp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Target:       &item,
			Destination:  &dropDest,
		}

		da := &DropActivity{}
		err = da.DoTurn(dropComp, player, world)
		require.NoError(t, err)

		assert.True(t, world.Components.LocationOnField.Has(item))
		gridElement := world.Components.GridElement.Get(item)
		assert.Equal(t, 9, int(gridElement.X))
		assert.Equal(t, 9, int(gridElement.Y))
		// 通常アイテムはPropコンポーネントを持たない
		assert.False(t, world.Components.Prop.Has(item))
	})
}
