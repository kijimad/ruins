package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/rawschema"
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

		useItemActivity := &useItemBehavior

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

		useItemActivity := &useItemBehavior

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

		useItemActivity := &useItemBehavior

		// 50の満腹度回復で95%以上になる
		err := useItemActivity.applyNutrition(comp, actor, world, 50, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		assert.Equal(t, 475, hungerComp.Current)
		assert.Equal(t, rawschema.HungerStageIdSatiated, hungerComp.GetLevel(world.Resources.NeedDefs), "満腹状態になっているはず")
	})

	t.Run("Hungerコンポーネントがない場合は何もしない", func(t *testing.T) {
		t.Parallel()

		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()
		// Hungerコンポーネントを追加しない

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)

		useItemActivity := &useItemBehavior

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

		useItemActivity := &useItemBehavior

		assert.Equal(t, rawschema.HungerStageIdStarving, hunger.GetLevel(world.Resources.NeedDefs), "初期状態は飢餓状態")

		// 300の満腹度回復で70%になる
		err := useItemActivity.applyNutrition(comp, actor, world, 300, item)
		require.NoError(t, err)

		hungerComp := world.Components.Hunger.Get(actor)
		require.NotNil(t, hungerComp)
		assert.Equal(t, 350, hungerComp.Current)
		assert.Equal(t, rawschema.HungerStageIdNormal, hungerComp.GetLevel(world.Resources.NeedDefs), "普通状態に回復しているはず")
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
			BehaviorName: gc.BehaviorEat,
			State:        gc.ActivityStateRunning,
			Params:       &gc.UseItemParams{Target: bread},
		}

		ua := &eatBehavior
		err := ua.DoTurn(comp, actor, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)

		// 満腹度が回復していることを確認
		hungerComp := world.Components.Hunger.Get(actor)
		assert.Equal(t, 350, hungerComp.Current)

		// パンが1個消費され、バックパックに2個残る。反復中の構造変更で panic しないことも兼ねる
		assert.Equal(t, 2, countBackpackByRawID(world, actor, "bread"))
	})

	t.Run("自傷ダメージアイテムを使用して使用者にダメージが入る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		world.Components.HP.Add(actor, &gc.HP{Current: 100, Max: 100})

		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "毒薬"})
		world.Components.RawID.Add(item, &gc.RawID{ID: "poison"})
		world.Components.InflictsDamage.Add(item, &gc.InflictsDamage{Amount: 10})
		world.Components.Consumable.Add(item, &gc.Consumable{})
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: actor})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
			Params:       &gc.UseItemParams{Target: item},
		}

		ua := &useItemBehavior
		err := ua.DoTurn(comp, actor, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Equal(t, 90, world.Components.HP.Get(actor).Current, "自傷ダメージぶんHPが減る")
		assert.Equal(t, 0, countBackpackByRawID(world, actor, "poison"), "使用したアイテムは消費される")
	})

	t.Run("Targetがnilの場合はエラーを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		world.Components.HP.Add(actor, &gc.HP{Current: 100, Max: 100})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
		}

		ua := &useItemBehavior
		err := ua.DoTurn(comp, actor, world)

		require.Error(t, err)
		assert.Equal(t, gc.ActivityStateRunning, comp.State, "中断は呼び出し側が行う")
	})

	t.Run("回復アイテムを使用してHPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)
		hp := world.Components.HP.Get(player)
		hp.Max = 100
		hp.Current = 50

		item, err := lifecycle.SpawnBackpackItem(world, "green_herb", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
			Params:       &gc.UseItemParams{Target: item},
		}

		ua := &useItemBehavior
		err = ua.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		assert.Equal(t, 81, world.Components.HP.Get(player).Current, "回復量ぶんHPが増える")
	})

	t.Run("治療アイテムを使用して不調が治る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)
		world.Components.HealthStatus.Get(player).Parts[gc.BodyPartLegs].SetCondition(
			gc.HealthCondition{Type: gc.ConditionType("fracture"), Timer: 60, Severity: gc.TimerToSeverity(60)})

		item, err := lifecycle.SpawnBackpackItem(world, "splint", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			State:        gc.ActivityStateRunning,
			Params:       &gc.UseItemParams{Target: item},
		}

		ua := &useItemBehavior
		err = ua.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
		leg := world.Components.HealthStatus.Get(player).Parts[gc.BodyPartLegs].GetCondition(gc.ConditionType("fracture"))
		require.NotNil(t, leg)
		assert.Equal(t, consts.Percent(100), leg.TendQuality, "治療アイテムの使用で不調が治療される")
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

		ua := &useItemBehavior
		err := ua.Validate(comp, actor, world)
		assert.NoError(t, err)
	})

	t.Run("空腹度回復アイテムは使用前検証を通る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)

		item, err := lifecycle.SpawnBackpackItem(world, "almond", 1)
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorEat,
			Params:       &gc.UseItemParams{Target: item},
		}

		ua := &eatBehavior
		err = ua.Validate(comp, player, world)
		assert.NoError(t, err)
	})

	t.Run("自傷ダメージアイテムは使用前検証を通る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
		require.NoError(t, err)

		// 自傷ダメージのみを持つraw項目は存在しないため、効果コンポーネントのみ手組みする
		item := world.ECS.NewEntity()
		world.Components.Consumable.Add(item, &gc.Consumable{})
		world.Components.InflictsDamage.Add(item, &gc.InflictsDamage{Amount: 10})

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorUseItem,
			Params:       &gc.UseItemParams{Target: item},
		}

		ua := &useItemBehavior
		err = ua.Validate(comp, player, world)
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

		ua := &useItemBehavior
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

		ua := &useItemBehavior
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

		ua := &useItemBehavior
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

		ua := &useItemBehavior
		err := ua.Validate(comp, actor, world)
		assert.Error(t, err)
	})
}

func TestUseItemBehavior_Info(t *testing.T) {
	t.Parallel()

	ua := &useItemBehavior
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

			u := &useItemBehavior
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

	u := &useItemBehavior
	// amount=1 は腐敗3割で整数除算すると0。最低1にクランプされ、栄養効果がゼロにならない
	require.NoError(t, u.applyNutrition(NewActivity(gc.BehaviorUseItem, 1), actor, world, 1, bread))

	assert.Equal(t, 101, world.Components.Hunger.Get(actor).Current, "腐敗でも最低1は回復する")
}

func TestUseItemBehavior_applyNutrition_腐敗食は食中毒を起こしうる(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		seed         uint64 // NewPCG(seed,0) の初回 IntN(100)。25 未満で発症する
		wantContract bool
	}{
		{"低い目で発症する", 4, true},   // 初回 11 < 25
		{"高い目で発症しない", 1, false}, // 初回 59 >= 25
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			bread, err := lifecycle.SpawnFieldItem(world, "bread", 5, 5, 1)
			require.NoError(t, err)
			query.GetGameTime(world).TotalTurns = 3000 // 腐敗
			world.Resources.Config.RNG = rand.New(rand.NewPCG(tt.seed, 0))

			// Player を付けないのでログ経路を避けつつ、HealthStatus で発症先を用意する
			actor := world.ECS.NewEntity()
			world.Components.Hunger.Add(actor, gc.NewHunger())
			world.Components.HealthStatus.Add(actor, &gc.HealthStatus{})

			u := &useItemBehavior
			require.NoError(t, u.applyNutrition(NewActivity(gc.BehaviorUseItem, 1), actor, world, 30, bread))

			cond := world.Components.HealthStatus.Get(actor).Parts[gc.BodyPartTorso].GetCondition(gc.ConditionType("food_poisoning"))
			if tt.wantContract {
				require.NotNil(t, cond, "腐敗食で食中毒を発症する")
			} else {
				assert.Nil(t, cond, "閾値を超えた目では発症しない")
			}
		})
	}
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

			u := &useItemBehavior
			require.NoError(t, u.applyNutrition(NewActivity(gc.BehaviorUseItem, 1), actor, world, 30, bread))

			recent := query.GetGameLog(world).GetRecent(1)
			require.Len(t, recent, 1)
			assert.Contains(t, recent[0], tt.want, "食べたログに鮮度が出る")
		})
	}
}

func TestUseItemBehavior_getItemName(t *testing.T) {
	t.Parallel()

	t.Run("Nameコンポーネントがあれば訳された名前を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "パン"})

		u := &useItemBehavior
		assert.Equal(t, "パン", u.getItemName(item, world))
	})

	t.Run("Nameコンポーネントがなければ汎用名を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		item := world.ECS.NewEntity() // Nameコンポーネントなし

		u := &useItemBehavior
		assert.Equal(t, "Item", u.getItemName(item, world))
	})
}

func TestUseItemBehavior_Name(t *testing.T) {
	t.Parallel()

	ua := &useItemBehavior
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

		ua := &useItemBehavior
		err := ua.applyHealing(comp, actor, world, healing, item)
		require.NoError(t, err)

		hp := world.Components.HP.Get(actor)
		assert.Equal(t, 80, hp.Current, "指定量ぶんHPが回復するべき")
	})

	t.Run("回復スキルの回復効果倍率が適用される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		actor := world.ECS.NewEntity()
		world.Components.HP.Add(actor, &gc.HP{Current: 10, Max: 100})
		// 回復Lv20: 回復効果 = 100 + 20*5 = 200
		skills := gc.NewSkills()
		skills.Get(rawschema.SkillIdHealing).Value = 20
		world.Components.Skills.Add(actor, skills)

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)
		healing := &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 20}

		ua := &useItemBehavior
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

		item := world.ECS.NewEntity()
		comp := NewActivity(gc.BehaviorUseItem, 1)
		healing := &gc.ProvidesHealing{Kind: gc.HealNumeral, Amount: 0}

		ua := &useItemBehavior
		err := ua.applyHealing(comp, actor, world, healing, item)
		require.NoError(t, err)

		hp := world.Components.HP.Get(actor)
		assert.Equal(t, 11, hp.Current, "回復量0でも最低1は回復するべき")
	})
}

func TestUseItemBehavior_logItemUse(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤー以外はログを出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity() // Playerコンポーネントなし
		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "薬"})

		u := &useItemBehavior
		u.logItemUse(actor, world, item, 10, true)

		assert.Empty(t, query.GetGameLog(world).GetRecent(1), "プレイヤー以外はログを出さない")
	})

	t.Run("回復のとき回復量がログに出る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "薬"})

		u := &useItemBehavior
		u.logItemUse(actor, world, item, 25, true)

		recent := query.GetGameLog(world).GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "recovered")
		assert.Contains(t, recent[0], "25")
	})

	t.Run("自傷ダメージのときダメージ量がログに出る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		actor := world.ECS.NewEntity()
		world.Components.Player.Add(actor, &gc.Player{})
		item := world.ECS.NewEntity()
		world.Components.Name.Add(item, &gc.Name{Name: "毒薬"})

		u := &useItemBehavior
		u.logItemUse(actor, world, item, 10, false)

		recent := query.GetGameLog(world).GetRecent(1)
		require.Len(t, recent, 1)
		assert.Contains(t, recent[0], "damage")
		assert.Contains(t, recent[0], "10")
	})
}

func TestUseItem_キューブの核を使うと隣にキューブが組み上がる(t *testing.T) {
	t.Parallel()
	surrounded := []consts.Coord[consts.Tile]{
		{X: 9, Y: 9}, {X: 10, Y: 9}, {X: 11, Y: 9}, {X: 9, Y: 10},
		{X: 11, Y: 10}, {X: 9, Y: 11}, {X: 10, Y: 11}, {X: 11, Y: 11},
	}
	far := consts.Coord[consts.Tile]{X: 15, Y: 15}
	above := consts.Coord[consts.Tile]{X: 10, Y: 9}
	right := consts.Coord[consts.Tile]{X: 11, Y: 10}
	tests := []struct {
		name         string
		overworld    bool
		cubes        []consts.Coord[consts.Tile]
		items        []consts.Coord[consts.Tile]
		wantCubes    []consts.Coord[consts.Tile]
		wantCoreLeft bool
		wantLog      i18n.Msgid
	}{
		{"地上でキューブが無ければ真上のマスに1台組み上がる", true, nil, nil, []consts.Coord[consts.Tile]{above}, false, "A new cube has been assembled."},
		{"地上でキューブが居ても2台目が組み上がる", true, []consts.Coord[consts.Tile]{far}, nil, []consts.Coord[consts.Tile]{far, above}, false, "A new cube has been assembled."},
		{"真上にキューブが居ればそのマスを飛ばす", true, []consts.Coord[consts.Tile]{above}, nil, []consts.Coord[consts.Tile]{above, right}, false, "A new cube has been assembled."},
		{"落ちている物のあるマスは飛ばす", true, nil, []consts.Coord[consts.Tile]{above}, []consts.Coord[consts.Tile]{right}, false, "A new cube has been assembled."},
		{"周りが埋まっていれば組み上がらず核も残る", true, nil, surrounded, nil, true, "There is no room to assemble the cube."},
		{"地上でなければ組み上がらず核も残る", false, nil, nil, nil, true, "The cube can only be assembled outdoors."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			if tt.overworld {
				*query.EnsureSeamlessBand(world) = gc.SeamlessBand{Active: true, ChunkW: 20, ChunkH: 20, Cols: 1, Rows: 1}
			}
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
			require.NoError(t, err)
			for _, c := range tt.cubes {
				_, err := lifecycle.SpawnCube(world, c)
				require.NoError(t, err)
			}
			for _, c := range tt.items {
				_, err := lifecycle.SpawnFieldItem(world, "iron", c.X, c.Y, 1)
				require.NoError(t, err)
			}
			core, err := lifecycle.SpawnFieldItem(world, "cube_core", 10, 10, 1)
			require.NoError(t, err)
			require.NoError(t, lifecycle.MoveToBackpack(world, core, player))

			_, err = Execute(NewUseItemActivity(core), player, world)
			require.NoError(t, err)

			assert.ElementsMatch(t, tt.wantCubes, query.CubeTiles(world))
			assert.Equal(t, tt.wantCoreLeft, world.ECS.Alive(core))
			entries := query.GetGameLog(world).GetRecentEntries(1)
			require.Len(t, entries, 1)
			assert.Equal(t, query.T(world, tt.wantLog), entries[0].Text())
		})
	}
}

func TestUseItem_満腹なら食べずにログを出す(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		current   int
		wantEaten bool
	}{
		{"満腹度が上限なら食べず食べ物も残る", gc.DefaultMaxHunger, false},
		{"上限より1でも少なければ食べて上限で頭打ちになる", gc.DefaultMaxHunger - 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
			require.NoError(t, err)
			world.Components.Hunger.Get(player).Current = tt.current
			item, err := lifecycle.SpawnBackpackItem(world, "almond", 1)
			require.NoError(t, err)

			result, err := Execute(NewEatActivity(item), player, world)
			require.NoError(t, err)

			assert.Equal(t, tt.wantEaten, result.Success)
			assert.Equal(t, !tt.wantEaten, world.ECS.Alive(item))
			assert.Equal(t, gc.DefaultMaxHunger, world.Components.Hunger.Get(player).Current)
			if !tt.wantEaten {
				entries := query.GetGameLog(world).GetRecentEntries(1)
				require.Len(t, entries, 1)
				assert.Equal(t, query.T(world, "Too full to eat"), entries[0].Text())
			}
		})
	}
}

func TestGetBehavior_食べるは食べるの組で動く(t *testing.T) {
	t.Parallel()

	b, err := GetBehavior(gc.BehaviorEat)
	require.NoError(t, err)
	ub, ok := b.(*UseItemBehavior)
	require.Truef(t, ok, "GetBehavior(BehaviorEat) は *UseItemBehavior を返すべきだが %T だった", b)
	assert.Equal(t, gc.BehaviorEat, ub.Name())
	assert.Equal(t, "Eat", ub.Info().Name)
}

func TestEffectSets_食べると使うの組は重ならない(t *testing.T) {
	t.Parallel()

	for _, e := range eatEffects {
		assert.NotContains(t, useEffects, e, "%T が両方の組にある", e)
	}
}

func TestUseItemBehavior_Validate_食べるの組の効果を持たない物は食べられない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	bandage, err := lifecycle.SpawnBackpackItem(world, "bandage", 1)
	require.NoError(t, err)

	ub := &eatBehavior
	require.ErrorIs(t, ub.Validate(NewEatActivity(bandage), player, world), errItemNoEffect)
}
