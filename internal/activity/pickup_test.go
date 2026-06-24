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

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		_, err = worldhelper.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Destination:  &destination,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("対象タイルにアイテムがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// アイテムは別のタイルにある
		_, err = worldhelper.SpawnFieldItem(world, "木刀", 20, 20, 1)
		require.NoError(t, err)

		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Destination:  &destination,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "拾えるものがありません")
	})

	t.Run("Destinationがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "目的地が指定されていません")
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

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := worldhelper.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Destination:  &destination,
		}

		pa := &PickupActivity{}
		err = pa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// アイテムがバックパックに移動していることを確認
		assert.True(t, item.HasComponent(world.Components.LocationInBackpack))
		// フィールドから消えていることを確認
		assert.False(t, item.HasComponent(world.Components.GridElement))
	})

	t.Run("対象タイルにアイテムがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		// 別のタイルにアイテムがある
		_, err = worldhelper.SpawnFieldItem(world, "木刀", 20, 20, 1)
		require.NoError(t, err)

		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Destination:  &destination,
		}

		pa := &PickupActivity{}
		err = pa.DoTurn(comp, player, world)

		assert.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("Destinationがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
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
}

func TestPickupActivity_DoTurn_Target(t *testing.T) {
	t.Parallel()

	t.Run("Targetが指定されている場合はそのアイテムだけを拾う", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item1, err := worldhelper.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		item2, err := worldhelper.SpawnFieldItem(world, "回復薬", 10, 10, 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Target:       &item1,
		}

		pa := &PickupActivity{}
		err = pa.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// 指定したアイテムだけがバックパックに移動する
		assert.True(t, item1.HasComponent(world.Components.LocationInBackpack))
		assert.False(t, item1.HasComponent(world.Components.GridElement))

		// 指定していないアイテムはフィールドに残る
		assert.False(t, item2.HasComponent(world.Components.LocationInBackpack))
		assert.True(t, item2.HasComponent(world.Components.GridElement))
	})
}

func TestPickupActivity_Validate_Target(t *testing.T) {
	t.Parallel()

	t.Run("Targetが拾得可能なら成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		item, err := worldhelper.SpawnFieldItem(world, "木刀", 10, 10, 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Target:       &item,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("TargetがPropの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "テストProp"})
		prop.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		prop.AddComponent(world.Components.LocationOnField, &gc.LocationOnField{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Target:       &prop,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "拾えるものがありません")
	})
}

func TestPickupActivity_Validate_Prop(t *testing.T) {
	t.Parallel()

	t.Run("Propは拾えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
		require.NoError(t, err)

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "テストProp"})
		prop.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
		prop.AddComponent(world.Components.LocationOnField, &gc.LocationOnField{})

		destination := gc.GridElement{X: 10, Y: 10}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Destination:  &destination,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "拾えるものがありません")
	})

	t.Run("アイテムとPropが同じタイルにある場合も拾える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
		require.NoError(t, err)

		_, err = worldhelper.SpawnFieldItem(world, "木刀", 5, 5, 1)
		require.NoError(t, err)
		// Interactableを持つPropも同じタイルにある
		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "テストProp"})
		prop.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
		prop.AddComponent(world.Components.Interactable, &gc.Interactable{Interactions: []gc.InteractionData{gc.MeleeInteraction{}}})

		destination := gc.GridElement{X: 5, Y: 5}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			Destination:  &destination,
		}

		pa := &PickupActivity{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})
}

func TestPickupActivity_DoTurn_Prop(t *testing.T) {
	t.Parallel()

	t.Run("Propのみのタイルでは拾得に失敗する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := worldhelper.SpawnPlayer(world, 8, 6, "Ash")
		require.NoError(t, err)

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "テストProp"})
		prop.AddComponent(world.Components.HP, &gc.HP{Max: 10, Current: 10})
		prop.AddComponent(world.Components.GridElement, &gc.GridElement{X: 8, Y: 6})
		prop.AddComponent(world.Components.LocationOnField, &gc.LocationOnField{})

		destination := gc.GridElement{X: 8, Y: 6}
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
			State:        gc.ActivityStateRunning,
			Destination:  &destination,
		}

		pa := &PickupActivity{}
		err = pa.DoTurn(comp, player, world)

		assert.Error(t, err, "Propは設置物なので拾えない")
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}
