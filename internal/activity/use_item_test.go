package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUseItemActivity_applyNutrition(t *testing.T) {
	t.Parallel()

	t.Run("満腹度が正常に増加する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.World.NewEntity()

		// Hungerコンポーネントを追加（DefaultMaxHunger = 500）
		hunger := gc.NewHunger()
		hunger.Current = 250 // 半分の満腹度
		world.Components.Hunger.Add(actor, hunger)

		item := world.World.NewEntity()
		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		// 100の満腹度回復
		err = useItemActivity.applyNutrition(comp, actor, world, 100, item)
		require.NoError(t, err)

		// 満腹度が250 + 100 = 350になっているはず
		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		updatedHunger := hungerComp
		assert.Equal(t, 350, updatedHunger.Current, "満腹度が正しく増加していない")
	})

	t.Run("満腹度が上限を超えない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.World.NewEntity()

		hunger := gc.NewHunger()
		hunger.Current = 475 // ほぼ満腹（500の95%）
		world.Components.Hunger.Add(actor, hunger)

		item := world.World.NewEntity()
		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		// 100の満腹度回復（上限を超える）
		err = useItemActivity.applyNutrition(comp, actor, world, 100, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		updatedHunger := hungerComp
		assert.Equal(t, gc.DefaultMaxHunger, updatedHunger.Current, "満腹度が上限を超えている")
	})

	t.Run("満腹状態になった場合", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.World.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})

		hunger := gc.NewHunger()
		hunger.Current = 425 // 85%（500の85%）
		world.Components.Hunger.Add(actor, hunger)

		item := world.World.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "パン"})

		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		// 50の満腹度回復で95%以上になる
		err = useItemActivity.applyNutrition(comp, actor, world, 50, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		updatedHunger := hungerComp
		assert.Equal(t, 475, updatedHunger.Current)
		assert.Equal(t, gc.HungerSatiated, updatedHunger.GetLevel(), "満腹状態になっているはず")
	})

	t.Run("Hungerコンポーネントがない場合は何もしない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.World.NewEntity()
		// Hungerコンポーネントを追加しない

		item := world.World.NewEntity()
		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		// エラーにならずに完了する
		err = useItemActivity.applyNutrition(comp, actor, world, 200, item)
		assert.NoError(t, err)
	})

	t.Run("飢餓状態から回復する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.World.NewEntity()

		hunger := gc.NewHunger()
		hunger.Current = 50 // 10%（500の10%）- 飢餓状態
		world.Components.Hunger.Add(actor, hunger)

		item := world.World.NewEntity()
		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		assert.Equal(t, gc.HungerStarving, hunger.GetLevel(), "初期状態は飢餓状態")

		// 300の満腹度回復で70%になる
		err = useItemActivity.applyNutrition(comp, actor, world, 300, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		updatedHunger := hungerComp
		assert.Equal(t, 350, updatedHunger.Current)
		assert.Equal(t, gc.HungerNormal, updatedHunger.GetLevel(), "普通状態に回復しているはず")
	})
}

func TestUseItemActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("空腹度回復アイテムを使用して完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.World.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		world.Components.HP.Add(actor, &gc.HP{Current: 100, Max: 100})
		hunger := gc.NewHunger()
		hunger.Current = 250
		world.Components.Hunger.Add(actor, hunger)

		item := world.World.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "パン"})
		world.Components.ProvidesNutrition.Add(item, &gc.ProvidesNutrition{Amount: 100})
		world.Components.Consumable.Add(item, &gc.Consumable{})
		world.Components.Stackable.Add(item, &gc.Stackable{Count: 3})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
			Target:       &item,
		}

		ua := &UseItemActivity{}
		err := ua.DoTurn(comp, actor, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// 満腹度が回復していることを確認
		hungerComp := world.Components.Hunger.Get(actor)
		assert.Equal(t, 350, hungerComp.Current)

		// アイテムが1つ消費されていることを確認
		stackableComp := world.Components.Stackable.Get(item)
		assert.Equal(t, 2, stackableComp.Count)
	})

	t.Run("Targetがnilの場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.World.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		world.Components.HP.Add(actor, &gc.HP{Current: 100, Max: 100})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
			Target:       nil,
		}

		ua := &UseItemActivity{}
		err := ua.DoTurn(comp, actor, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestUseItemActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効なアイテムの場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.World.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		world.Components.HP.Add(actor, &gc.HP{Current: 100, Max: 100})

		item := world.World.NewEntity()
		world.Components.Consumable.Add(item, &gc.Consumable{})
		world.Components.ProvidesHealing.Add(item, &gc.ProvidesHealing{Amount: gc.NumeralAmount{Numeral: 50}})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       &item,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		assert.NoError(t, err)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.World.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       nil,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		require.Error(t, err)
		assert.Equal(t, ErrItemNotSet, err)
	})

	t.Run("効果コンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.World.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{})

		item := world.World.NewEntity()

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       &item,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		require.Error(t, err)
		assert.Equal(t, ErrItemNoEffect, err)
	})

	t.Run("効果がないアイテムの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.World.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{})

		item := world.World.NewEntity()
		world.Components.Material.Add(item, &gc.Material{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       &item,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		require.Error(t, err)
		assert.Equal(t, ErrItemNoEffect, err)
	})

	t.Run("ActorにHPがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.World.NewEntity()
		// HPなし

		item := world.World.NewEntity()
		world.Components.Consumable.Add(item, &gc.Consumable{})
		world.Components.ProvidesHealing.Add(item, &gc.ProvidesHealing{Amount: gc.NumeralAmount{Numeral: 50}})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       &item,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		require.Error(t, err)
		assert.Equal(t, ErrActorNoHP, err)
	})
}

func TestUseItemActivity_Info(t *testing.T) {
	t.Parallel()

	ua := &UseItemActivity{}
	info := ua.Info()

	assert.Equal(t, "アイテム使用", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}
