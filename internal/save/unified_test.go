package save

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaveLoadRoundTrip はセーブ・ロードでワールド全体のデータが保存・復元されることを検証する。
// ark-serde の出力順序は非決定的（Types配列はマップ反復順）なため、JSON文字列一致ではなく
// 復元後のデータで検証する。
func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("JSON文字列経由でデータが保存・復元される", func(t *testing.T) {
		t.Parallel()
		originalWorld := createComplexDeterministicWorld(t)
		sm := createTestSerializationManager(t)

		jsonStr, err := sm.GenerateWorldJSON(originalWorld)
		require.NoError(t, err)

		newWorld := testutil.InitTestWorld(t)
		require.NoError(t, sm.RestoreWorldFromJSON(newWorld, jsonStr))

		assertComplexWorldRestored(t, newWorld)
	})

	t.Run("ファイル経由でデータが保存・復元される", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		sm, err := NewSerializationManager(WithSaveDir(tempDir))
		require.NoError(t, err)
		originalWorld := createComplexDeterministicWorld(t)

		require.NoError(t, sm.SaveWorld(originalWorld, "original"))

		loadedWorld := testutil.InitTestWorld(t)
		require.NoError(t, sm.LoadWorld(loadedWorld, "original"))

		assertComplexWorldRestored(t, loadedWorld)
	})

	t.Run("多段階のセーブ・ロードでデータが保たれる", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		sm, err := NewSerializationManager(WithSaveDir(tempDir))
		require.NoError(t, err)
		world := createComplexDeterministicWorld(t)

		for cycle := range 3 {
			filename := "cycle_" + string(rune('0'+cycle))
			require.NoError(t, sm.SaveWorld(world, filename))
			newWorld := testutil.InitTestWorld(t)
			require.NoError(t, sm.LoadWorld(newWorld, filename))
			assertComplexWorldRestored(t, newWorld)
			world = newWorld
		}
	})
}

// assertComplexWorldRestored は createComplexDeterministicWorld のデータが復元されていることを検証する
func assertComplexWorldRestored(t *testing.T, world w.World) {
	t.Helper()

	// プレイヤーを取得し、基本値まで復元されることを検証する
	var playerEntity ecs.Entity
	playerCount := 0
	pq := ecs.NewFilter1[gc.Player](world.ECS).Query()
	for pq.Next() {
		playerCount++
		playerEntity = pq.Entity()
	}
	require.Equal(t, 1, playerCount, "プレイヤーが1体復元される")
	require.True(t, world.ECS.Alive(playerEntity), "復元後のプレイヤーが生存している")
	assert.Equal(t, "テストプレイヤー", world.Components.Name.Get(playerEntity).Name)
	assert.Equal(t, 100, world.Components.HP.Get(playerEntity).Max, "HPが値まで復元される")
	assert.Equal(t, 9, world.Components.Abilities.Get(playerEntity).Agility.Total, "能力値が復元される")

	// 敵NPCが3体（丸ごと保存で復元される）
	npcCount := 0
	nq := ecs.NewFilter1[gc.FactionEnemyData](world.ECS).Query()
	for nq.Next() {
		npcCount++
	}
	assert.Equal(t, 3, npcCount, "敵NPCが3体復元される")

	// バックパック内アイテムを名前で引きつつ、エンティティ参照(Owner)の再マッピングを検証する。
	// ark-serde は復元時にエンティティ参照を張り替えるため、全アイテムの Owner が
	// 復元後プレイヤーを指す（かつ生存する）ことを確認する。ここが壊れると所有関係が静かに崩壊する
	items := map[string]ecs.Entity{}
	iq := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for iq.Next() {
		e := iq.Entity()
		owner := world.Components.LocationInBackpack.Get(e).Owner
		require.True(t, world.ECS.Alive(owner), "アイテムのOwnerが生存エンティティを指す")
		assert.Equal(t, playerEntity, owner, "アイテムのOwnerが復元後プレイヤーへ再マッピングされる")
		if world.Components.Name.Has(e) {
			items[world.Components.Name.Get(e).Name] = e
		}
	}

	// 武器・防具・スタック可能アイテムが値まで復元される
	sword, ok := items["木刀"]
	require.True(t, ok, "木刀が復元される")
	assert.Equal(t, 8, world.Components.Melee.Get(sword).Damage, "近接武器のダメージが復元される")

	armor, ok := items["西洋鎧"]
	require.True(t, ok, "西洋鎧が復元される")
	assert.Equal(t, 15, world.Components.Wearable.Get(armor).Defense, "防具の防御力が復元される")

	iron, ok := items["鉄"]
	require.True(t, ok, "マテリアルが復元される")
	assert.True(t, world.Components.Stackable.Has(iron), "Stackableが復元される")

	// 回復薬（ProvidesHealing 倍率0.3）が値まで復元される
	healRatioFound := false
	hq := ecs.NewFilter1[gc.ProvidesHealing](world.ECS).Query()
	for hq.Next() {
		ph := world.Components.ProvidesHealing.Get(hq.Entity())
		if ph.Kind == gc.HealRatio && ph.Ratio == 0.3 {
			healRatioFound = true
		}
	}
	assert.True(t, healRatioFound, "回復薬のProvidesHealing(倍率0.3)が復元される")
}

// createTestSerializationManager テスト用のSerializationManagerを作成
func createTestSerializationManager(t *testing.T) *SerializationManager {
	t.Helper()
	tempDir := t.TempDir()
	sm, err := NewSerializationManager(WithSaveDir(tempDir))
	require.NoError(t, err)
	return sm
}

// createComplexDeterministicWorld InitNewGameDataのような複雑なワールドを作成する
func createComplexDeterministicWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)

	// プレイヤー作成（手動でコンポーネント追加）
	player := world.ECS.NewEntity()
	world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.FactionAlly.Add(player, &gc.FactionAllyData{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: consts.Tile(10), Y: consts.Tile(15)})
	world.Components.Abilities.Add(player, &gc.Abilities{
		Vitality:  gc.Ability{Base: 10, Modifier: 0, Total: 10},
		Strength:  gc.Ability{Base: 8, Modifier: 0, Total: 8},
		Sensation: gc.Ability{Base: 6, Modifier: 0, Total: 6},
		Dexterity: gc.Ability{Base: 7, Modifier: 0, Total: 7},
		Agility:   gc.Ability{Base: 9, Modifier: 0, Total: 9},
		Defense:   gc.Ability{Base: 5, Modifier: 0, Total: 5},
	})
	world.Components.HP.Add(player, &gc.HP{Current: 100, Max: 100})
	world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{})

	// 武器1: 木刀
	sword := world.ECS.NewEntity()
	world.Components.Name.Add(sword, &gc.Name{Name: "木刀"})
	world.Components.LocationInBackpack.Add(sword, &gc.LocationInBackpack{Owner: player})
	world.Components.Melee.Add(sword, &gc.Melee{
		Accuracy: 100, Damage: 8, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
	})

	// 武器2: ハンドガン
	handgun := world.ECS.NewEntity()
	world.Components.Name.Add(handgun, &gc.Name{Name: "ハンドガン"})
	world.Components.LocationInBackpack.Add(handgun, &gc.LocationInBackpack{Owner: player})
	world.Components.Melee.Add(handgun, &gc.Melee{
		Accuracy: 85, Damage: 12, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackHandgun,
	})

	// 防具: 西洋鎧
	armor := world.ECS.NewEntity()
	world.Components.Name.Add(armor, &gc.Name{Name: "西洋鎧"})
	world.Components.LocationInBackpack.Add(armor, &gc.LocationInBackpack{Owner: player})
	world.Components.Wearable.Add(armor, &gc.Wearable{
		Defense:           15,
		EquipmentCategory: gc.EquipmentTorso,
		EquipBonus: gc.EquipBonus{
			Vitality: 2, Strength: 1, Sensation: 0, Dexterity: 0, Agility: -1,
		},
	})

	// 回復アイテム
	potion := world.ECS.NewEntity()
	world.Components.Name.Add(potion, &gc.Name{Name: "回復薬"})
	world.Components.LocationInBackpack.Add(potion, &gc.LocationInBackpack{Owner: player})
	world.Components.Consumable.Add(potion, &gc.Consumable{
		UsableScene: gc.UsableSceneAny,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupAlly,
			TargetNum:   gc.TargetSingle,
		},
	})
	world.Components.ProvidesHealing.Add(potion, &gc.ProvidesHealing{
		Kind:  gc.HealRatio,
		Ratio: 0.3,
	})

	// NPC作成
	for i := range 3 {
		npc := world.ECS.NewEntity()
		world.Components.Name.Add(npc, &gc.Name{Name: "NPC" + string(rune('A'+i))})
		world.Components.GridElement.Add(npc, &gc.GridElement{
			X: consts.Tile(20 + i*5),
			Y: consts.Tile(25 + i*3),
		})
		world.Components.SoloAI.Add(npc, &gc.SoloAI{ViewDistance: 5})
		world.Components.FactionEnemy.Add(npc, &gc.FactionEnemyData{})
		world.Components.Abilities.Add(npc, &gc.Abilities{
			Vitality:  gc.Ability{Base: 10 + i, Modifier: 0, Total: 10 + i},
			Strength:  gc.Ability{Base: 8 + i, Modifier: 0, Total: 8 + i},
			Sensation: gc.Ability{Base: 6 + i, Modifier: 0, Total: 6 + i},
			Dexterity: gc.Ability{Base: 7 + i, Modifier: 0, Total: 7 + i},
			Agility:   gc.Ability{Base: 9 + i, Modifier: 0, Total: 9 + i},
			Defense:   gc.Ability{Base: 5 + i, Modifier: 0, Total: 5 + i},
		})
		world.Components.HP.Add(npc, &gc.HP{Current: 100 + i*10, Max: 100 + i*10})
		world.Components.WeightCapacity.Add(npc, &gc.WeightCapacity{})
	}

	// マテリアル
	material1 := world.ECS.NewEntity()
	world.Components.Name.Add(material1, &gc.Name{Name: "鉄"})
	world.Components.Value.Add(material1, &gc.Value{})
	world.Components.LocationInBackpack.Add(material1, &gc.LocationInBackpack{Owner: player})
	world.Components.Stackable.Add(material1, &gc.Stackable{})

	material2 := world.ECS.NewEntity()
	world.Components.Name.Add(material2, &gc.Name{Name: "緑ハーブ"})
	world.Components.Value.Add(material2, &gc.Value{})
	world.Components.LocationInBackpack.Add(material2, &gc.LocationInBackpack{Owner: player})
	world.Components.Stackable.Add(material2, &gc.Stackable{})

	return world
}
