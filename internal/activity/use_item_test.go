package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countBackpackByRawID は player のバックパックにある id のアイテム数を数える
func countBackpackByRawID(world w.World, player ecs.Entity, id string) int {
	n := 0
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationInBackpack.Get(e).Owner == player && world.Components.RawID.Get(e).ID == id {
			n++
		}
	}
	return n
}

func TestUseItemBehavior_applyNutrition(t *testing.T) {
	t.Parallel()

	t.Run("満腹度が正常に増加する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()

		// Hungerコンポーネントを追加（DefaultMaxHunger = 500）
		hunger := gc.NewHunger()
		hunger.Current = 250 // 半分の満腹度
		world.Components.Hunger.Add(actor, hunger)

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)

		useItemActivity := &UseItemBehavior{}

		// 100の満腹度回復
		err := useItemActivity.applyNutrition(comp, actor, world, 100, item)
		require.NoError(t, err)

		// 満腹度が250 + 100 = 350になっているはず
		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		assert.Equal(t, 350, hungerComp.Current, "満腹度が正しく増加していない")
	})

	t.Run("満腹度が上限を超えない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()

		hunger := gc.NewHunger()
		hunger.Current = 475 // ほぼ満腹（500の95%）
		world.Components.Hunger.Add(actor, hunger)

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)

		useItemActivity := &UseItemBehavior{}

		// 100の満腹度回復（上限を超える）
		err := useItemActivity.applyNutrition(comp, actor, world, 100, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		assert.Equal(t, gc.DefaultMaxHunger, hungerComp.Current, "満腹度が上限を超えている")
	})

	t.Run("満腹状態になった場合", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})

		hunger := gc.NewHunger()
		hunger.Current = 425 // 85%（500の85%）
		world.Components.Hunger.Add(actor, hunger)

		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "パン"})

		comp := NewActivity(gc.BehaviorUseItem, 1)

		useItemActivity := &UseItemBehavior{}

		// 50の満腹度回復で95%以上になる
		err := useItemActivity.applyNutrition(comp, actor, world, 50, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		assert.Equal(t, 475, hungerComp.Current)
		assert.Equal(t, gc.HungerSatiated, hungerComp.GetLevel(), "満腹状態になっているはず")
	})

	t.Run("Hungerコンポーネントがない場合は何もしない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()
		// Hungerコンポーネントを追加しない

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)

		useItemActivity := &UseItemBehavior{}

		// エラーにならずに完了する
		err := useItemActivity.applyNutrition(comp, actor, world, 200, item)
		assert.NoError(t, err)
	})

	t.Run("飢餓状態から回復する", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()

		hunger := gc.NewHunger()
		hunger.Current = 50 // 10%（500の10%）- 飢餓状態
		world.Components.Hunger.Add(actor, hunger)

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)

		useItemActivity := &UseItemBehavior{}

		assert.Equal(t, gc.HungerStarving, hunger.GetLevel(), "初期状態は飢餓状態")

		// 300の満腹度回復で70%になる
		err := useItemActivity.applyNutrition(comp, actor, world, 300, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		assert.Equal(t, 350, hungerComp.Current)
		assert.Equal(t, gc.HungerNormal, hungerComp.GetLevel(), "普通状態に回復しているはず")
	})
}

func TestUseItemBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("空腹度回復アイテムを使用して完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		world.Components.HP.Add(actor, &gc.HP{Current: 100, Max: 100})
		hunger := gc.NewHunger()
		hunger.Current = 250
		world.Components.Hunger.Add(actor, hunger)

		// パンを3個 actor のバックパックへ置く。1個1エンティティなので3エンティティ作る
		var bread ecs.Entity
		for range 3 {
			e := world.ECS.NewEntity()
			world.Components.Name.Add(e, &gc.Name{Name: "パン"})
			world.Components.RawID.Add(e, &gc.RawID{ID: "bread"})
			world.Components.ProvidesNutrition.Add(e, &gc.ProvidesNutrition{Amount: 100})
			world.Components.Consumable.Add(e, &gc.Consumable{})
			world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: actor})
			bread = e
		}

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
			Params:       &gc.UseItemParams{Target: bread},
		}

		ua := &UseItemBehavior{}
		err := ua.DoTurn(comp, actor, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// 満腹度が回復していることを確認
		hungerComp := world.Components.Hunger.Get(actor)
		assert.Equal(t, 350, hungerComp.Current)

		// パンが1個消費され、バックパックに2個残る。反復中の構造変更で panic しないことも兼ねる
		assert.Equal(t, 2, countBackpackByRawID(world, actor, "bread"))
	})

	t.Run("Targetがnilの場合はキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		world.Components.HP.Add(actor, &gc.HP{Current: 100, Max: 100})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
		}

		ua := &UseItemBehavior{}
		err := ua.DoTurn(comp, actor, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestUseItemBehavior_Validate(t *testing.T) {
	t.Parallel()

	t.Run("有効なアイテムの場合は成功", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		world.Components.HP.Add(actor, &gc.HP{Current: 100, Max: 100})

		item := world.ECS.NewEntity()
		world.Components.Consumable.Add(item, &gc.Consumable{})
		world.Components.ProvidesHealing.Add(item, &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 50})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Params:       &gc.UseItemParams{Target: item},
		}

		ua := &UseItemBehavior{}
		err := ua.Validate(comp, actor, world)
		assert.NoError(t, err)
	})

	t.Run("Targetがnilの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
		}

		ua := &UseItemBehavior{}
		err := ua.Validate(comp, actor, world)
		assert.ErrorIs(t, err, ErrParamsTypeMismatch)
	})

	t.Run("効果コンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Params:       &gc.UseItemParams{Target: world.ECS.NewEntity()},
		}

		ua := &UseItemBehavior{}
		err := ua.Validate(comp, actor, world)
		require.Error(t, err)
		var ve *UserError
		require.NotErrorAs(t, err, &ve)
	})

	t.Run("効果がないアイテムの場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{})

		item := world.ECS.NewEntity()

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Params:       &gc.UseItemParams{Target: item},
		}

		ua := &UseItemBehavior{}
		err := ua.Validate(comp, actor, world)
		require.Error(t, err)
		var ve *UserError
		require.NotErrorAs(t, err, &ve)
	})

	t.Run("ActorにHPがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		// HPなし

		item := world.ECS.NewEntity()
		world.Components.Consumable.Add(item, &gc.Consumable{})
		world.Components.ProvidesHealing.Add(item, &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 50})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Params:       &gc.UseItemParams{Target: item},
		}

		ua := &UseItemBehavior{}
		err := ua.Validate(comp, actor, world)
		assert.Error(t, err)
	})
}

func TestUseItemBehavior_Info(t *testing.T) {
	t.Parallel()

	ua := &UseItemBehavior{}
	info := ua.Info()

	assert.Equal(t, "Use Item", info.Name)
	assert.False(t, info.Interruptible)
	assert.False(t, info.Resumable)
}

func TestUseItemBehavior_applyNutrition_鮮度で栄養が変わる(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		now         consts.Turn // 生成は TotalTurns=0。bread の StageLength は 1500
		startHunger int
		wantHunger  int
	}{
		{"新鮮は満額", 0, 250, 280},       // +30
		{"劣化は半減", 1500, 250, 265},    // +15
		{"腐敗は3割の栄養", 3000, 250, 259}, // +9
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			bread, err := lifecycle.SpawnFieldItem(world, "bread", 5, 5, 1)
			require.NoError(t, err)
			query.GetGameTime(world).TotalTurns = tt.now

			actor := world.ECS.NewEntity()
			hunger := gc.NewHunger()
			hunger.Current = tt.startHunger
			world.Components.Hunger.Add(actor, hunger)

			u := &UseItemBehavior{}
			comp := NewActivity(gc.BehaviorUseItem, 1)
			require.NoError(t, u.applyNutrition(comp, actor, world, 30, bread))

			assert.Equal(t, tt.wantHunger, world.Components.Hunger.Get(actor).Current)
		})
	}
}

func TestUseItemBehavior_applyNutrition_鮮度で減っても最低1は与える(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	bread, err := lifecycle.SpawnFieldItem(world, "bread", 5, 5, 1)
	require.NoError(t, err)
	query.GetGameTime(world).TotalTurns = 3000 // 腐敗

	actor := world.ECS.NewEntity()
	hunger := gc.NewHunger()
	hunger.Current = 100
	world.Components.Hunger.Add(actor, hunger)

	u := &UseItemBehavior{}
	// amount=1 は腐敗3割で整数除算すると0。最低1にクランプされ、栄養効果がゼロにならない
	require.NoError(t, u.applyNutrition(NewActivity(gc.BehaviorUseItem, 1), actor, world, 1, bread))

	assert.Equal(t, 101, world.Components.Hunger.Get(actor).Current, "腐敗でも最低1は回復する")
}

func TestUseItemBehavior_食べたログに鮮度が出る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		now  consts.Turn // 生成は TotalTurns=0。bread の StageLength は 1500
		want string
	}{
		{"新鮮", 0, "fresh"},
		{"劣化", 1500, "old"},
		{"腐敗", 3000, "spoiled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			bread, err := lifecycle.SpawnFieldItem(world, "bread", 5, 5, 1)
			require.NoError(t, err)
			query.GetGameTime(world).TotalTurns = tt.now

			actor := world.ECS.NewEntity()
			world.Components.Player.Add(actor, &gc.Player{})
			hunger := gc.NewHunger()
			hunger.Current = 100
			world.Components.Hunger.Add(actor, hunger)

			u := &UseItemBehavior{}
			require.NoError(t, u.applyNutrition(NewActivity(gc.BehaviorUseItem, 1), actor, world, 30, bread))

			recent := query.GetGameLog(world).GetRecent(1)
			require.Len(t, recent, 1)
			assert.Contains(t, recent[0], tt.want, "食べたログに鮮度が出る")
		})
	}
}

func TestUseItemBehavior_Name(t *testing.T) {
	t.Parallel()

	ua := &UseItemBehavior{}
	assert.Equal(t, gc.BehaviorUseItem, ua.Name())
}

func TestNewUseItemActivity(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	item := world.ECS.NewEntity()

	comp := NewUseItemActivity(item)

	assert.Equal(t, gc.BehaviorUseItem, comp.BehaviorName)
	params, ok := comp.Params.(*gc.UseItemParams)
	require.True(t, ok, "UseItemParamsが設定されるべき")
	assert.Equal(t, item, params.Target)
}

func TestUseItemBehavior_applyHealing(t *testing.T) {
	t.Parallel()

	t.Run("絶対量指定の回復がHPに反映される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{Current: 50, Max: 100})

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)
		healing := &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 30}

		ua := &UseItemBehavior{}
		err := ua.applyHealing(comp, actor, world, healing, item)
		require.NoError(t, err)

		hp := world.Components.HP.Get(actor)
		assert.Equal(t, 80, hp.Current, "指定量ぶんHPが回復するべき")
	})

	t.Run("CharModifiersの回復効果倍率が適用される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{Current: 10, Max: 100})
		world.Components.CharModifiers.Add(actor, &gc.CharModifiers{HealingEffect: 200})

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)
		healing := &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 20}

		ua := &UseItemBehavior{}
		err := ua.applyHealing(comp, actor, world, healing, item)
		require.NoError(t, err)

		hp := world.Components.HP.Get(actor)
		assert.Equal(t, 50, hp.Current, "倍率2倍で40回復し、10+40=50になるべき")
	})

	t.Run("回復量が1未満になる場合は最低1にクランプされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{Current: 10, Max: 100})
		world.Components.CharModifiers.Add(actor, &gc.CharModifiers{HealingEffect: 0})

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)
		healing := &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 30}

		ua := &UseItemBehavior{}
		err := ua.applyHealing(comp, actor, world, healing, item)
		require.NoError(t, err)

		hp := world.Components.HP.Get(actor)
		assert.Equal(t, 11, hp.Current, "倍率0でも最低1は回復するべき")
	})
}
