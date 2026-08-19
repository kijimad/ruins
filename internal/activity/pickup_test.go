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

func TestPickupBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("同じタイルにアイテムがある場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnFieldItem(world, "wooden_sword", 10, 10, 1)
		require.NoError(t, err)

		comp := NewPickupTileActivity(world, consts.Coord[consts.Tile]{X: 10, Y: 10})

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("対象タイルにアイテムがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// アイテムは別のタイルにある
		_, err = lifecycle.SpawnFieldItem(world, "wooden_sword", 20, 20, 1)
		require.NoError(t, err)

		// 対象タイルには拾えるものがないので Targets が空になる
		comp := NewPickupTileActivity(world, consts.Coord[consts.Tile]{X: 10, Y: 10})

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		var ve *UserError
		require.ErrorAs(t, err, &ve)
	})

	t.Run("パラメータがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// PickupParams が付かない不正なアクティビティは型アサートで弾かれる
		comp := &gc.Activity{
			BehaviorName: gc.BehaviorPickup,
		}

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		require.ErrorIs(t, err, ErrParamsTypeMismatch)
	})
}

func TestPickupBehavior_Info(t *testing.T) {
	t.Parallel()

	pa := &PickupBehavior{}
	info := pa.Info()

	assert.Equal(t, "Pick Up", info.Name)
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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 10, 10, 1)
		require.NoError(t, err)

		comp := NewPickupTileActivity(world, consts.Coord[consts.Tile]{X: 10, Y: 10})

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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// 別のタイルにアイテムがある
		_, err = lifecycle.SpawnFieldItem(world, "wooden_sword", 20, 20, 1)
		require.NoError(t, err)

		comp := NewPickupTileActivity(world, consts.Coord[consts.Tile]{X: 10, Y: 10})

		pa := &PickupBehavior{}
		err = pa.DoTurn(comp, player, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("パラメータがない場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		// PickupParams が付かない不正なアクティビティは DoTurn でキャンセルへ落ちる
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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item1, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 10, 10, 1)
		require.NoError(t, err)

		item2, err := lifecycle.SpawnFieldItem(world, "healing_potion", 10, 10, 1)
		require.NoError(t, err)

		// item1 のスタックだけを対象にする。別品種の item2 は別スタックなので残る
		comp := NewPickupStackActivity(world, item1)

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

func TestNewPickupStackActivity_同一スタックをまとめて拾う(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// 同一品種3個を同じタイルへ。代表1個を対象にスタック丸ごと拾う
	rep, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 10, 10, 3)
	require.NoError(t, err)
	require.Equal(t, 3, query.GetEntityCount(world, rep), "拾う前は床に3個")

	comp := NewPickupStackActivity(world, rep)
	pa := &PickupBehavior{}
	require.NoError(t, pa.DoTurn(comp, player, world))
	assert.Equal(t, gc.ActivityStateCompleted, comp.State)

	// 代表を含むスタック全部がバックパックへ移る。表示の個数と拾う個数が揃う
	assert.Equal(t, 3, query.GetEntityCount(world, rep), "3個ともバックパックのスタックになる")
	assert.True(t, world.Components.LocationInBackpack.Has(rep))
	assert.Empty(t, query.PickablesAt(world, consts.Coord[consts.Tile]{X: 10, Y: 10}), "床に拾える物は残らない")
}

func TestPickupBehavior_Validate_Target(t *testing.T) {
	t.Parallel()

	t.Run("Targetが拾得可能なら成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnFieldItem(world, "wooden_sword", 10, 10, 1)
		require.NoError(t, err)

		comp := NewPickupStackActivity(world, item)

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("Targetが固定物の場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.LocationOnField.Add(prop, &gc.LocationOnField{})

		comp := NewPickupStackActivity(world, prop)

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		var ve *UserError
		require.ErrorAs(t, err, &ve)
	})
}

func TestPickupBehavior_Validate_Fixed(t *testing.T) {
	t.Parallel()

	t.Run("固定物は拾えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.LocationOnField.Add(prop, &gc.LocationOnField{})

		// 固定物は PickablesAt で除かれ Targets が空になる
		comp := NewPickupTileActivity(world, consts.Coord[consts.Tile]{X: 10, Y: 10})

		pa := &PickupBehavior{}
		err = pa.Validate(comp, player, world)
		var ve *UserError
		require.ErrorAs(t, err, &ve)
	})

	t.Run("アイテムと固定物が同じタイルにある場合も拾える", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)

		_, err = lifecycle.SpawnFieldItem(world, "wooden_sword", 5, 5, 1)
		require.NoError(t, err)
		// Interactableを持つ固定物も同じタイルにある
		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})
		world.Components.Interactable.Add(prop, &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionMelee}})

		comp := NewPickupTileActivity(world, consts.Coord[consts.Tile]{X: 5, Y: 5})

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

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 8, Y: 6}, "ash")
		require.NoError(t, err)

		prop := world.ECS.NewEntity()
		world.Components.Fixed.Add(prop, &gc.Fixed{})
		world.Components.Name.Add(prop, &gc.Name{Name: "テスト固定物"})
		world.Components.HP.Add(prop, &gc.HP{Max: 10, Current: 10})
		world.Components.GridElement.Add(prop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 8, Y: 6}})
		world.Components.LocationOnField.Add(prop, &gc.LocationOnField{})

		comp := NewPickupTileActivity(world, consts.Coord[consts.Tile]{X: 8, Y: 6})

		pa := &PickupBehavior{}
		err = pa.DoTurn(comp, player, world)

		require.Error(t, err, "固定物は拾えない")
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}
