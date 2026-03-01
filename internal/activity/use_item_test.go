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
		actor := world.Manager.NewEntity()

		// Hungerコンポーネントを追加（DefaultMaxHunger = 500）
		hunger := gc.NewHunger()
		hunger.Current = 250 // 半分の満腹度
		actor.AddComponent(world.Components.Hunger, hunger)

		item := world.Manager.NewEntity()
		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		// 100の満腹度回復
		err = useItemActivity.applyNutrition(comp, actor, world, 100, item)
		require.NoError(t, err)

		// 満腹度が250 + 100 = 350になっているはず
		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		updatedHunger := hungerComp.(*gc.Hunger)
		assert.Equal(t, 350, updatedHunger.Current, "満腹度が正しく増加していない")
	})

	t.Run("満腹度が上限を超えない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.Manager.NewEntity()

		hunger := gc.NewHunger()
		hunger.Current = 475 // ほぼ満腹（500の95%）
		actor.AddComponent(world.Components.Hunger, hunger)

		item := world.Manager.NewEntity()
		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		// 100の満腹度回復（上限を超える）
		err = useItemActivity.applyNutrition(comp, actor, world, 100, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		updatedHunger := hungerComp.(*gc.Hunger)
		assert.Equal(t, gc.DefaultMaxHunger, updatedHunger.Current, "満腹度が上限を超えている")
	})

	t.Run("満腹状態になった場合", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.Player, &gc.Player{})

		hunger := gc.NewHunger()
		hunger.Current = 425 // 85%（500の85%）
		actor.AddComponent(world.Components.Hunger, hunger)

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})

		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		// 50の満腹度回復で95%以上になる
		err = useItemActivity.applyNutrition(comp, actor, world, 50, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		updatedHunger := hungerComp.(*gc.Hunger)
		assert.Equal(t, 475, updatedHunger.Current)
		assert.Equal(t, gc.HungerSatiated, updatedHunger.GetLevel(), "満腹状態になっているはず")
	})

	t.Run("Hungerコンポーネントがない場合は何もしない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.Manager.NewEntity()
		// Hungerコンポーネントを追加しない

		item := world.Manager.NewEntity()
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
		actor := world.Manager.NewEntity()

		hunger := gc.NewHunger()
		hunger.Current = 50 // 10%（500の10%）- 飢餓状態
		actor.AddComponent(world.Components.Hunger, hunger)

		item := world.Manager.NewEntity()
		comp, err := NewActivity(&UseItemActivity{}, 1)
		require.NoError(t, err)

		useItemActivity := &UseItemActivity{}

		assert.Equal(t, gc.HungerStarving, hunger.GetLevel(), "初期状態は飢餓状態")

		// 300の満腹度回復で70%になる
		err = useItemActivity.applyNutrition(comp, actor, world, 300, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		updatedHunger := hungerComp.(*gc.Hunger)
		assert.Equal(t, 350, updatedHunger.Current)
		assert.Equal(t, gc.HungerNormal, updatedHunger.GetLevel(), "普通状態に回復しているはず")
	})
}

func TestUseItemActivity_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("空腹度回復アイテムを使用して完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.Player, &gc.Player{})
		actor.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 100, Max: 100},
		})
		hunger := gc.NewHunger()
		hunger.Current = 250
		actor.AddComponent(world.Components.Hunger, hunger)

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 3})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		item.AddComponent(world.Components.ProvidesNutrition, &gc.ProvidesNutrition{Amount: 100})
		item.AddComponent(world.Components.Consumable, &gc.Consumable{})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{})

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
		hungerComp := world.Components.Hunger.Get(actor).(*gc.Hunger)
		assert.Equal(t, 350, hungerComp.Current)

		// アイテムが1つ消費されていることを確認
		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 2, itemComp.Count)
	})

	t.Run("Targetがnilの場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.Player, &gc.Player{})
		actor.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 100, Max: 100},
		})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
			Target:       nil,
		}

		ua := &UseItemActivity{}
		err := ua.DoTurn(comp, actor, world)

		assert.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestUseItemActivity_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効なアイテムの場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.Player, &gc.Player{})
		actor.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 100, Max: 100},
		})

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{})
		item.AddComponent(world.Components.ProvidesHealing, &gc.ProvidesHealing{Amount: gc.NumeralAmount{Numeral: 50}})

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

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.Pools, &gc.Pools{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       nil,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		assert.Error(t, err)
		assert.Equal(t, ErrItemNotSet, err)
	})

	t.Run("Itemコンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.Pools, &gc.Pools{})

		item := world.Manager.NewEntity()
		// Itemコンポーネントなし

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       &item,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidItem, err)
	})

	t.Run("効果がないアイテムの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		actor.AddComponent(world.Components.Pools, &gc.Pools{})

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{})
		// 効果コンポーネントなし

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       &item,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		assert.Error(t, err)
		assert.Equal(t, ErrItemNoEffect, err)
	})

	t.Run("ActorにPoolsがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.Manager.NewEntity()
		// Poolsなし

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{})
		item.AddComponent(world.Components.ProvidesHealing, &gc.ProvidesHealing{Amount: gc.NumeralAmount{Numeral: 50}})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Target:       &item,
		}

		ua := &UseItemActivity{}
		err := ua.Validate(comp, actor, world)
		assert.Error(t, err)
		assert.Equal(t, ErrActorNoPools, err)
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
