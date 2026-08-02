package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickupBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("同じタイルにアイテムがある場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Params:       &gc.PlaceParams{Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("対象タイルにアイテムがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		// アイテムは別のタイルにある
		_, err = lifecycle.SpawnFieldItem(world, "木刀", 20, 20, 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Params:       &gc.PlaceParams{Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "拾えるものがありません")
	})

	t.Run("パラメータがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		// 値型パラメータでは目的地未指定を表せない。拾得対象も拾得先も無い不正なアクティビティは
		// PlaceParams が付いていないため弾かれる
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
		}

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "拾得対象が指定されていません")
	})
}

func TestPickupBehavior_Info(t *testing.T) {
	t.Parallel()

	pa := &PickupBehavior{}
	info := pa.Info()

	assert.Equal(t, "拾得", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestPickupBehavior_Name(t *testing.T) {
	t.Parallel()

	pa := &PickupBehavior{}
	assert.Equal(t, gc.BehaviorPickup, pa.Name())
}

func TestPickupBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("正常にアイテムを拾って完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Params:       &gc.PlaceParams{Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		pa := &PickupBehavior{}
		err = pa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// アイテムがバックパックに移動していることを確認
		assert.True(t, world.Components.LocationInBackpack.Has(item))
		// フィールドから消えていることを確認
		assert.False(t, world.Components.GridElement.Has(item))
	})

	t.Run("対象タイルにアイテムがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		// 別のタイルにアイテムがある
		_, err = lifecycle.SpawnFieldItem(world, "木刀", 20, 20, 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Params:       &gc.PlaceParams{Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		pa := &PickupBehavior{}
		err = pa.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("Destinationがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
		}

		pa := &PickupBehavior{}
		err = pa.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestPickupBehavior_DoTurn_Target(t *testing.T) {
	t.Parallel()

	t.Run("Targetが指定されている場合はそのアイテムだけを拾う", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item1, err := lifecycle.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		item2, err := lifecycle.SpawnFieldItem(world, "回復薬", 10, 10, 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Params:       &gc.PlaceParams{Target: item1},
		}

		pa := &PickupBehavior{}
		err = pa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// 指定したアイテムだけがバックパックに移動する
		assert.True(t, world.Components.LocationInBackpack.Has(item1))
		assert.False(t, world.Components.GridElement.Has(item1))

		// 指定していないアイテムはフィールドに残る
		assert.False(t, world.Components.LocationInBackpack.Has(item2))
		assert.True(t, world.Components.GridElement.Has(item2))
	})
}

func TestPickupBehavior_Validate_Target(t *testing.T) {
	t.Parallel()

	t.Run("Targetが拾得可能なら成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Params:       &gc.PlaceParams{Target: item},
		}

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Targetが固定物の場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.LocationOnField.Add(prop, &gc.LocationOnField{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Params:       &gc.PlaceParams{Target: prop},
		}

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "拾えるものがありません")
	})
}

func TestPickupBehavior_Validate_Fixed(t *testing.T) {
	t.Parallel()

	t.Run("固定物は拾えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.LocationOnField.Add(prop, &gc.LocationOnField{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Params:       &gc.PlaceParams{Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}},
		}

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "拾えるものがありません")
	})

	t.Run("アイテムと固定物が同じタイルにある場合も拾える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnFieldItem(world, "木刀", 5, 5, 1)
		require.NoError(t, err)
		// Interactableを持つ固定物も同じタイルにある
		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
		world.Components.Interactable.Add(prop, &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionMelee}})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Params:       &gc.PlaceParams{Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}},
		}

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})
}

func TestPickupBehavior_DoTurn_Fixed(t *testing.T) {
	t.Parallel()

	t.Run("固定物のみのタイルでは拾得に失敗する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 8, Y: 6}, "Ash")
		require.NoError(t, err)

		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.HP.Add(prop, &gc.HP{Max: 10, Current: 10})
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 8, Y: 6}})
		world.Components.LocationOnField.Add(prop, &gc.LocationOnField{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Params:       &gc.PlaceParams{Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 8, Y: 6}}},
		}

		pa := &PickupBehavior{}
		err = pa.DoTurn(comp, player, world)

		require.Error(t, err, "固定物は拾えない")
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}
