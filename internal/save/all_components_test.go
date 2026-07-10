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
	player := world.World.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.FactionAlly.Add(player, &gc.FactionAllyData{})
	world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})
	world.Components.Description.Add(player, &gc.Description{Description: "説明文"})
	world.Components.HP.Add(player, &gc.HP{Current: 80, Max: 100})
	world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{Current: 12.5, Max: 50.0})
	world.Components.TurnBased.Add(player, &gc.TurnBased{
		AP:    gc.IntPool{Current: 3, Max: 5},
		Speed: 12,
	})
	world.Components.Abilities.Add(player, &gc.Abilities{
		Vitality:  gc.Ability{Base: 10, Modifier: 2, Total: 12},
		Strength:  gc.Ability{Base: 15, Modifier: -1, Total: 14},
		Sensation: gc.Ability{Base: 8, Modifier: 0, Total: 8},
		Dexterity: gc.Ability{Base: 12, Modifier: 3, Total: 15},
		Agility:   gc.Ability{Base: 9, Modifier: 1, Total: 10},
		Defense:   gc.Ability{Base: 7, Modifier: 0, Total: 7},
	})
	world.Components.Camera.Add(player, &gc.Camera{
		Scale: 2.0, ScaleTo: 2.5,
		X: 100.0, Y: 200.0, TargetX: 150.0, TargetY: 250.0,
	})
	world.Components.GridElement.Add(player, &gc.GridElement{X: consts.Tile(5), Y: consts.Tile(10)})
	world.Components.SpriteRender.Add(player, &gc.SpriteRender{
		SpriteSheetName: "player_sheet",
		SpriteKey:       "idle",
		Depth:           gc.DepthNum(10),
		AnimKeys:        []string{"walk_left", "walk_right"},
	})
	world.Components.LightSource.Add(player, &gc.LightSource{
		Radius:  consts.Tile(3),
		Enabled: true,
		Color:   color.RGBA{R: 255, G: 200, B: 100, A: 128},
	})
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 9999})

	// === 近接武器（バックパック内） ===
	sword := world.World.NewEntity()
	world.Components.Name.Add(sword, &gc.Name{Name: "炎の剣"})
	world.Components.Description.Add(sword, &gc.Description{Description: "炎属性の剣"})
	world.Components.LocationInBackpack.Add(sword, &gc.LocationInBackpack{Owner: player})
	world.Components.Value.Add(sword, &gc.Value{Value: 500})
	world.Components.Melee.Add(sword, &gc.Melee{
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
	gun := world.World.NewEntity()
	world.Components.Name.Add(gun, &gc.Name{Name: "ハンドガン"})
	world.Components.LocationInBackpack.Add(gun, &gc.LocationInBackpack{Owner: player})
	world.Components.Fire.Add(gun, &gc.Fire{
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
	ammo := world.World.NewEntity()
	world.Components.Name.Add(ammo, &gc.Name{Name: "9mm弾"})
	world.Components.LocationInBackpack.Add(ammo, &gc.LocationInBackpack{Owner: player})
	world.Components.Stackable.Add(ammo, &gc.Stackable{})
	world.Components.Ammo.Add(ammo, &gc.Ammo{
		AmmoTag:       "9mm",
		DamageBonus:   5,
		AccuracyBonus: 10,
	})

	// === 防具（装備中） ===
	armor := world.World.NewEntity()
	world.Components.Name.Add(armor, &gc.Name{Name: "鋼の鎧"})
	world.Components.LocationEquipped.Add(armor, &gc.LocationEquipped{
		Owner:         player,
		EquipmentSlot: gc.EquipmentSlotNumber(1),
	})
	world.Components.StatsChanged.Add(armor, &gc.StatsChanged{})
	world.Components.Wearable.Add(armor, &gc.Wearable{
		Defense:           20,
		EquipmentCategory: gc.EquipmentTorso,
		EquipBonus: gc.EquipBonus{
			Vitality: 3, Strength: 1, Sensation: 0, Dexterity: -1, Agility: -2,
		},
		InsulationCold: 5,
		InsulationHeat: 2,
	})

	// === 回復アイテム: RatioAmount（バックパック内） ===
	potion := world.World.NewEntity()
	world.Components.Name.Add(potion, &gc.Name{Name: "回復薬"})
	world.Components.LocationInBackpack.Add(potion, &gc.LocationInBackpack{Owner: player})
	world.Components.Stackable.Add(potion, &gc.Stackable{})
	world.Components.Consumable.Add(potion, &gc.Consumable{
		UsableScene: gc.UsableSceneAny,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupAlly,
			TargetNum:   gc.TargetSingle,
		},
	})
	world.Components.ProvidesHealing.Add(potion, &gc.ProvidesHealing{
		Amount: gc.RatioAmount{Ratio: 0.5},
	})

	// === 回復アイテム: NumeralAmount（バックパック内） ===
	herb := world.World.NewEntity()
	world.Components.Value.Add(herb, &gc.Value{})
	world.Components.Name.Add(herb, &gc.Name{Name: "薬草"})
	world.Components.LocationInBackpack.Add(herb, &gc.LocationInBackpack{Owner: player})
	world.Components.Stackable.Add(herb, &gc.Stackable{})
	world.Components.ProvidesHealing.Add(herb, &gc.ProvidesHealing{
		Amount: gc.NumeralAmount{Numeral: 30},
	})

	// === 食料（バックパック内） ===
	food := world.World.NewEntity()
	world.Components.Value.Add(food, &gc.Value{})
	world.Components.Name.Add(food, &gc.Name{Name: "携帯食料"})
	world.Components.LocationInBackpack.Add(food, &gc.LocationInBackpack{Owner: player})
	world.Components.Stackable.Add(food, &gc.Stackable{})
	world.Components.ProvidesNutrition.Add(food, &gc.ProvidesNutrition{Amount: 50})

	// === ダメージアイテム（バックパック内） ===
	grenade := world.World.NewEntity()
	world.Components.Name.Add(grenade, &gc.Name{Name: "手榴弾"})
	world.Components.LocationInBackpack.Add(grenade, &gc.LocationInBackpack{Owner: player})
	world.Components.Consumable.Add(grenade, &gc.Consumable{
		UsableScene: gc.UsableSceneBattle,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupEnemy,
			TargetNum:   gc.TargetAll,
		},
	})
	world.Components.InflictsDamage.Add(grenade, &gc.InflictsDamage{Amount: 40})

	// === レシピ付きアイテム（バックパック内） ===
	craftable := world.World.NewEntity()
	world.Components.Value.Add(craftable, &gc.Value{})
	world.Components.Name.Add(craftable, &gc.Name{Name: "合成武器"})
	world.Components.LocationInBackpack.Add(craftable, &gc.LocationInBackpack{Owner: player})
	world.Components.Recipe.Add(craftable, &gc.Recipe{
		Inputs: []gc.RecipeInput{
			{Name: "鉄", Amount: 3},
			{Name: "木材", Amount: 1},
		},
	})

	// === 隊員エンティティ ===
	member := world.World.NewEntity()
	world.Components.Name.Add(member, &gc.Name{Name: "テスト隊員"})
	world.Components.HP.Add(member, &gc.HP{Current: 50, Max: 60})
	world.Components.Abilities.Add(member, &gc.Abilities{
		Vitality:  gc.Ability{Base: 8, Modifier: 0, Total: 8},
		Strength:  gc.Ability{Base: 10, Modifier: 0, Total: 10},
		Sensation: gc.Ability{Base: 6, Modifier: 0, Total: 6},
		Dexterity: gc.Ability{Base: 7, Modifier: 0, Total: 7},
		Agility:   gc.Ability{Base: 9, Modifier: 0, Total: 9},
		Defense:   gc.Ability{Base: 5, Modifier: 0, Total: 5},
	})
	world.Components.SquadMember.Add(member, &gc.SquadMember{})
	world.Components.SquadAI.Add(member, &gc.SquadAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SquadEscort,
		ItemPickup:    gc.PolicyPickup,
		ItemHandling:  gc.PolicyKeep,
	})
	world.Components.GridElement.Add(member, &gc.GridElement{X: consts.Tile(6), Y: consts.Tile(11)})
	world.Components.SpriteRender.Add(member, &gc.SpriteRender{
		SpriteSheetName: "npc_sheet",
		SpriteKey:       "idle",
		Depth:           gc.DepthNum(10),
	})
	world.Components.TurnBased.Add(member, &gc.TurnBased{
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
