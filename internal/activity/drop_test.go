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

func TestDropBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効なドロップ対象の場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Params:       &gc.PlaceParams{Target: item, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		da := &DropBehavior{}
		err = da.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
		}

		da := &DropBehavior{}
		err = da.Validate(comp, player, world)
		require.ErrorIs(t, err, ErrParamsTypeMismatch)
	})

	t.Run("バックパック内にないアイテムの場合は不変条件違反", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// バックパック外のアイテムを手動で作成
		item := world.ECS.NewEntity()
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Params:       &gc.PlaceParams{Target: item, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		da := &DropBehavior{}
		err = da.Validate(comp, player, world)
		require.Error(t, err)
		var ve *UserError
		require.NotErrorAs(t, err, &ve)
	})

	t.Run("パラメータがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 値型パラメータでは目的地未指定を表せない。ドロップ対象と目的地は構築関数が必ずまとめて渡す。
		// PlaceParams が付かない不正なアクティビティは弾かれる
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
		}

		da := &DropBehavior{}
		err = da.Validate(comp, player, world)
		require.ErrorIs(t, err, ErrParamsTypeMismatch)
	})
}

func TestNewDropActivity(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
	require.NoError(t, err)

	dest := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 4}}
	comp := NewDropActivity(item, dest)

	assert.Equal(t, gc.BehaviorDrop, comp.BehaviorName)
	params, ok := comp.Params.(*gc.PlaceParams)
	require.True(t, ok, "PlaceParamsが設定されるべき")
	assert.Equal(t, item, params.Target)
	assert.Equal(t, dest, params.Destination)
}

func TestDropBehavior_Info(t *testing.T) {
	t.Parallel()

	da := &DropBehavior{}
	info := da.Info()

	assert.Equal(t, "Drop", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestDropBehavior_Name(t *testing.T) {
	t.Parallel()

	da := &DropBehavior{}
	assert.Equal(t, gc.BehaviorDrop, da.Name())
}

func TestDropBehavior_performDrop(t *testing.T) {
	t.Parallel()

	t.Run("アイテムをフィールドにドロップできる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Params:       &gc.PlaceParams{Target: item, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		da := &DropBehavior{}
		err = da.performDrop(comp, player, world)
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
		assert.Contains(t, recent[0], "Dropped")
	})

	t.Run("スタックを丸ごと落とし個数ぶんフィールドに出す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 同一スタック3個をバックパックへ。代表1個をドロップ対象にする
		rep, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 3)
		require.NoError(t, err)
		require.Equal(t, 3, query.GetEntityCount(world, rep), "落とす前はスタック3個")

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Params:       &gc.PlaceParams{Target: rep, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		da := &DropBehavior{}
		require.NoError(t, da.performDrop(comp, player, world))

		// 同一タイルへ3個すべてが束ねて出る。ログの個数と落下数が一致する
		assert.Equal(t, 3, query.GetEntityCount(world, rep), "床の同タイルに3個出る")
		assert.True(t, world.Components.LocationOnField.Has(rep))

		// 同種はバックパックに残らない
		_, ok := query.FindStackInInventory(world, "wooden_sword")
		assert.False(t, ok, "スタック丸ごと落ちるのでバックパックに同種は残らない")

		// ログは個数3を表記する
		store := query.GetGameLog(world)
		recent := store.GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "3", "ログに落とした個数が出る")
	})

	t.Run("パラメータがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// PlaceParams が付かない不正なアクティビティは performDrop でも弾かれる
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
		}

		da := &DropBehavior{}
		err = da.performDrop(comp, player, world)
		require.ErrorIs(t, err, ErrParamsTypeMismatch)
	})
}

func TestDropBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常にドロップして完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Params:       &gc.PlaceParams{Target: item, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		da := &DropBehavior{}
		err = da.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("パラメータがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// PlaceParams が付かない不正なアクティビティは DoTurn でキャンセルへ落ちる
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
		}

		da := &DropBehavior{}
		err = da.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestDropBehavior_performDrop_AdjacentTile(t *testing.T) {
	t.Parallel()

	t.Run("隣接タイルにアイテムをドロップできる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		// プレイヤーの右隣にドロップ
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Params:       &gc.PlaceParams{Target: item, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}}},
		}

		da := &DropBehavior{}
		err = da.performDrop(comp, player, world)
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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		// 右下斜めにドロップ
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			Params:       &gc.PlaceParams{Target: item, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 11}}},
		}

		da := &DropBehavior{}
		err = da.performDrop(comp, player, world)
		require.NoError(t, err)

		gridElement := world.Components.GridElement.Get(item)
		assert.Equal(t, 11, int(gridElement.X))
		assert.Equal(t, 11, int(gridElement.Y))
	})
}

func TestDropBehavior_FixtureDerivedItem(t *testing.T) {
	t.Parallel()

	t.Run("固定物由来アイテムをドロップすると Fixed コンポーネントが保持される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
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
			Params:       &gc.PlaceParams{Target: prop, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}}},
		}

		da := &DropBehavior{}
		err = da.performDrop(comp, player, world)
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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 10, 10, 1)
		require.NoError(t, err)

		// 拾う。拾得対象は構築時にタイルから解決する
		pickupComp := NewPickupTileActivity(world, consts.Coord[consts.Tile]{X: 10, Y: 10})

		pa := &PickupBehavior{}
		err = pa.DoTurn(pickupComp, player, world)
		require.NoError(t, err)

		assert.True(t, world.Components.LocationInBackpack.Has(item))
		assert.False(t, world.Components.GridElement.Has(item))

		// ドロップ
		dropComp := &gc.Activity{
			BehaviorName: gc.BehaviorDrop,
			State:        gc.ActivityStateRunning,
			Params:       &gc.PlaceParams{Target: item, Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 9}}},
		}

		da := &DropBehavior{}
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
