package save

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAllComponentsGolden はセーブ対象の全コンポーネントとリソースを含むワールドを
// save→load→saveし、2回のJSON出力が一致することで網羅性を確認する。
// さらにゴールデンファイルと比較して、シリアライズ形式の意図しない変更を検知する。
func TestAllComponentsGolden(t *testing.T) {
	t.Parallel()
	sm := createTestSerializationManager(t)
	world := buildAllComponentsWorld(t)

	// 1回目のセーブ
	json1, err := sm.GenerateWorldJSON(world)
	require.NoError(t, err)

	// ロード
	newWorld := testutil.InitTestWorld(t)
	err = sm.RestoreWorldFromJSON(newWorld, json1)
	require.NoError(t, err)

	// 2回目のセーブ
	json2, err := sm.GenerateWorldJSON(newWorld)
	require.NoError(t, err)

	// ラウンドトリップ検証: save→load→saveでJSONが変化しないこと
	norm1 := normalizeJSONForComparison(json1)
	norm2 := normalizeJSONForComparison(json2)
	assert.Equal(t, norm1, norm2, "save→load→saveでJSONが変化した。コンポーネントの変換に欠落がある可能性がある")

	// ゴールデンファイルと比較
	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata"),
		goldie.WithNameSuffix(".golden.json"),
		goldie.WithDiffEngine(goldie.ColoredDiff),
	)
	g.Assert(t, "all_components", []byte(norm1))
}

// buildAllComponentsWorld はセーブ対象の全コンポーネントを含むワールドを構築する
func buildAllComponentsWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)

	// === プレイヤーエンティティ: マーカー + データコンポーネント ===
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.FactionAlly, &gc.FactionAllyData{})
	player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})
	player.AddComponent(world.Components.Description, &gc.Description{Description: "説明文"})
	player.AddComponent(world.Components.HP, &gc.HP{Current: 80, Max: 100})
	player.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{Current: 12.5, Max: 50.0})
	player.AddComponent(world.Components.TurnBased, &gc.TurnBased{
		AP:    gc.IntPool{Current: 3, Max: 5},
		Speed: 12,
	})
	player.AddComponent(world.Components.Abilities, &gc.Abilities{
		Vitality:  gc.Ability{Base: 10, Modifier: 2, Total: 12},
		Strength:  gc.Ability{Base: 15, Modifier: -1, Total: 14},
		Sensation: gc.Ability{Base: 8, Modifier: 0, Total: 8},
		Dexterity: gc.Ability{Base: 12, Modifier: 3, Total: 15},
		Agility:   gc.Ability{Base: 9, Modifier: 1, Total: 10},
		Defense:   gc.Ability{Base: 7, Modifier: 0, Total: 7},
	})
	player.AddComponent(world.Components.Camera, &gc.Camera{
		Scale: 2.0, ScaleTo: 2.5,
		X: 100.0, Y: 200.0, TargetX: 150.0, TargetY: 250.0,
	})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(5), Y: consts.Tile(10)})
	player.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{
		SpriteSheetName: "player_sheet",
		SpriteKey:       "idle",
		Depth:           gc.DepthNum(10),
		AnimKeys:        []string{"walk_left", "walk_right"},
	})
	player.AddComponent(world.Components.LightSource, &gc.LightSource{
		Radius:  consts.Tile(3),
		Enabled: true,
		Color:   color.RGBA{R: 255, G: 200, B: 100, A: 128},
	})
	player.AddComponent(world.Components.Wallet, &gc.Wallet{Currency: 9999})

	// === 近接武器（バックパック内） ===
	sword := world.Manager.NewEntity()
	sword.AddComponent(world.Components.Name, &gc.Name{Name: "炎の剣"})
	sword.AddComponent(world.Components.Description, &gc.Description{Description: "炎属性の剣"})
	sword.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})
	sword.AddComponent(world.Components.Value, &gc.Value{Value: 500})
	sword.AddComponent(world.Components.Melee, &gc.Melee{
		Accuracy:    90,
		Damage:      25,
		AttackCount: 2,
		Element:     gc.ElementTypeFire,
		AttackCategory: gc.AttackType{
			Type:  "SWORD",
			Range: gc.AttackRangeMelee,
			Label: "剣",
		},
		Cost: 3,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupEnemy,
			TargetNum:   gc.TargetSingle,
		},
	})

	// === 射撃武器（バックパック内） ===
	gun := world.Manager.NewEntity()
	gun.AddComponent(world.Components.Name, &gc.Name{Name: "ハンドガン"})
	gun.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})
	gun.AddComponent(world.Components.Fire, &gc.Fire{
		Accuracy:    80,
		Damage:      15,
		AttackCount: 1,
		Element:     gc.ElementTypeNone,
		AttackCategory: gc.AttackType{
			Type:  "HANDGUN",
			Range: gc.AttackRangeRanged,
			Label: "拳銃",
		},
		Cost: 2,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupEnemy,
			TargetNum:   gc.TargetSingle,
		},
		Magazine:            6,
		MagazineSize:        8,
		ReloadEffort:        3,
		AmmoTag:             "9mm",
		LoadedDamageBonus:   5,
		LoadedAccuracyBonus: 10,
	})

	// === 弾薬（バックパック内、スタック可能） ===
	ammo := world.Manager.NewEntity()
	ammo.AddComponent(world.Components.Name, &gc.Name{Name: "9mm弾"})
	ammo.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})
	ammo.AddComponent(world.Components.Stackable, &gc.Stackable{})
	ammo.AddComponent(world.Components.Ammo, &gc.Ammo{
		AmmoTag:       "9mm",
		DamageBonus:   5,
		AccuracyBonus: 10,
	})

	// === 防具（装備中） ===
	armor := world.Manager.NewEntity()
	armor.AddComponent(world.Components.Name, &gc.Name{Name: "鋼の鎧"})
	armor.AddComponent(world.Components.LocationEquipped, &gc.LocationEquipped{
		Owner:         player,
		EquipmentSlot: gc.EquipmentSlotNumber(1),
	})
	armor.AddComponent(world.Components.StatsChanged, &gc.StatsChanged{})
	armor.AddComponent(world.Components.Wearable, &gc.Wearable{
		Defense:           20,
		EquipmentCategory: gc.EquipmentTorso,
		EquipBonus: gc.EquipBonus{
			Vitality: 3, Strength: 1, Sensation: 0, Dexterity: -1, Agility: -2,
		},
		InsulationCold: 5,
		InsulationHeat: 2,
	})

	// === 回復アイテム: RatioAmount（バックパック内） ===
	potion := world.Manager.NewEntity()
	potion.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})
	potion.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})
	potion.AddComponent(world.Components.Stackable, &gc.Stackable{})
	potion.AddComponent(world.Components.Consumable, &gc.Consumable{
		UsableScene: gc.UsableSceneAny,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupAlly,
			TargetNum:   gc.TargetSingle,
		},
	})
	potion.AddComponent(world.Components.ProvidesHealing, &gc.ProvidesHealing{
		Amount: gc.RatioAmount{Ratio: 0.5},
	})

	// === 回復アイテム: NumeralAmount（バックパック内） ===
	herb := world.Manager.NewEntity()
	herb.AddComponent(world.Components.Value, &gc.Value{})
	herb.AddComponent(world.Components.Name, &gc.Name{Name: "薬草"})
	herb.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})
	herb.AddComponent(world.Components.Stackable, &gc.Stackable{})
	herb.AddComponent(world.Components.ProvidesHealing, &gc.ProvidesHealing{
		Amount: gc.NumeralAmount{Numeral: 30},
	})

	// === 食料（バックパック内） ===
	food := world.Manager.NewEntity()
	food.AddComponent(world.Components.Value, &gc.Value{})
	food.AddComponent(world.Components.Name, &gc.Name{Name: "携帯食料"})
	food.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})
	food.AddComponent(world.Components.Stackable, &gc.Stackable{})
	food.AddComponent(world.Components.ProvidesNutrition, &gc.ProvidesNutrition{Amount: 50})

	// === ダメージアイテム（バックパック内） ===
	grenade := world.Manager.NewEntity()
	grenade.AddComponent(world.Components.Name, &gc.Name{Name: "手榴弾"})
	grenade.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})
	grenade.AddComponent(world.Components.Consumable, &gc.Consumable{
		UsableScene: gc.UsableSceneBattle,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupEnemy,
			TargetNum:   gc.TargetAll,
		},
	})
	grenade.AddComponent(world.Components.InflictsDamage, &gc.InflictsDamage{Amount: 40})

	// === レシピ付きアイテム（バックパック内） ===
	craftable := world.Manager.NewEntity()
	craftable.AddComponent(world.Components.Value, &gc.Value{})
	craftable.AddComponent(world.Components.Name, &gc.Name{Name: "合成武器"})
	craftable.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{Owner: player})
	craftable.AddComponent(world.Components.Recipe, &gc.Recipe{
		Inputs: []gc.RecipeInput{
			{Name: "鉄", Amount: 3},
			{Name: "木材", Amount: 1},
		},
	})

	// === 隊員エンティティ ===
	member := world.Manager.NewEntity()
	member.AddComponent(world.Components.Name, &gc.Name{Name: "テスト隊員"})
	member.AddComponent(world.Components.HP, &gc.HP{Current: 50, Max: 60})
	member.AddComponent(world.Components.Abilities, &gc.Abilities{
		Vitality:  gc.Ability{Base: 8, Modifier: 0, Total: 8},
		Strength:  gc.Ability{Base: 10, Modifier: 0, Total: 10},
		Sensation: gc.Ability{Base: 6, Modifier: 0, Total: 6},
		Dexterity: gc.Ability{Base: 7, Modifier: 0, Total: 7},
		Agility:   gc.Ability{Base: 9, Modifier: 0, Total: 9},
		Defense:   gc.Ability{Base: 5, Modifier: 0, Total: 5},
	})
	member.AddComponent(world.Components.SquadMember, &gc.SquadMember{})
	member.AddComponent(world.Components.AI, &gc.AI{
		Planner:               gc.PlannerSquad,
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		Movement:              gc.MovementEscort,
		ItemPickup:            gc.PolicyPickup,
		ItemHandling:          gc.PolicyKeep,
		SubState:              gc.AIStateWaiting,
		DurationSubStateTurns: 2,
	})
	member.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(6), Y: consts.Tile(11)})
	member.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{
		SpriteSheetName: "npc_sheet",
		SpriteKey:       "idle",
		Depth:           gc.DepthNum(10),
	})
	member.AddComponent(world.Components.TurnBased, &gc.TurnBased{
		AP:    gc.IntPool{Current: 4, Max: 4},
		Speed: 10,
	})

	// === GameProgress リソース ===
	gp := query.GetGameProgress(world)
	gp.MarkDungeonCleared("遺跡")
	gp.MarkDungeonCleared("洞窟")
	gp.SetEventActive("boss_defeated")
	gp.MarkEventSeen("boss_defeated")

	return world
}
