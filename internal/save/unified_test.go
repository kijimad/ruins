package save

import (
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/mlange-42/ark/ecs"
)

// TestJSONDeterministicBehavior JSON出力の決定的動作を包括的にテスト
func TestJSONDeterministicBehavior(t *testing.T) {
	t.Parallel()

	t.Run("同一セッション内での安定性", func(t *testing.T) {
		t.Parallel()
		// 同じワールドインスタンスで複数回JSON生成
		world := createStandardTestWorld(t)
		sm := createTestSerializationManager(t)

		jsonStrings := make([]string, 0, 5)
		for range 5 {
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// すべて同一であることを確認
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		for i := 1; i < len(jsonStrings); i++ {
			normalizedJSON := normalizeJSONForComparison(jsonStrings[i])
			assert.JSONEq(t, baseJSON, normalizedJSON,
				"同じワールドから生成されたJSON %d が一致しません", i+1)
		}
	})

	t.Run("異なるセッション間での安定性", func(t *testing.T) {
		t.Parallel()
		// 異なるワールドインスタンスで同じデータを作成
		jsonStrings := make([]string, 0, 3)
		for range 3 {
			world := createStandardTestWorld(t)
			sm := createTestSerializationManager(t)
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// すべて同一であることを確認
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		for i := 1; i < len(jsonStrings); i++ {
			normalizedJSON := normalizeJSONForComparison(jsonStrings[i])
			assert.JSONEq(t, baseJSON, normalizedJSON,
				"セッション %d のJSONが一致しません", i+1)
		}
	})

	t.Run("コンポーネント追加順序に依存しない", func(t *testing.T) {
		t.Parallel()
		// 異なる順序でコンポーネントを追加したワールドを作成
		jsonStrings := make([]string, 0, 3)

		for variant := range 3 {
			world := testutil.InitTestWorld(t)

			entity := world.World.NewEntity()

			// バリアント毎に異なる順序でコンポーネントを追加
			switch variant {
			case 0:
				// 順序: Name -> GridElement -> Attack
				world.Components.Name.Add(entity, &gc.Name{Name: "テストエンティティ"})
				world.Components.GridElement.Add(entity, &gc.GridElement{X: consts.Tile(1), Y: consts.Tile(1)})
				world.Components.Melee.Add(entity, &gc.Melee{
					Accuracy: 85, Damage: 20, AttackCount: 1,
					Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
				})
			case 1:
				// 順序: Attack -> Name -> GridElement
				world.Components.Melee.Add(entity, &gc.Melee{
					Accuracy: 85, Damage: 20, AttackCount: 1,
					Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
				})
				world.Components.Name.Add(entity, &gc.Name{Name: "テストエンティティ"})
				world.Components.GridElement.Add(entity, &gc.GridElement{X: consts.Tile(1), Y: consts.Tile(1)})
			case 2:
				// 順序: GridElement -> Attack -> Name
				world.Components.GridElement.Add(entity, &gc.GridElement{X: consts.Tile(1), Y: consts.Tile(1)})
				world.Components.Melee.Add(entity, &gc.Melee{
					Accuracy: 85, Damage: 20, AttackCount: 1,
					Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
				})
				world.Components.Name.Add(entity, &gc.Name{Name: "テストエンティティ"})
			}

			sm := createTestSerializationManager(t)
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// すべてのバリアントが同じJSONを生成することを確認
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		for i := 1; i < len(jsonStrings); i++ {
			normalizedJSON := normalizeJSONForComparison(jsonStrings[i])
			assert.JSONEq(t, baseJSON, normalizedJSON,
				"コンポーネント追加順序による差異 (variant %d)", i+1)
		}
	})

	t.Run("エンティティ作成順序に依存しない", func(t *testing.T) {
		t.Parallel()
		// 異なる順序でエンティティを作成
		jsonStrings := make([]string, 0, 2)

		for variant := range 2 {
			world := testutil.InitTestWorld(t)

			entities := make([]ecs.Entity, 0, 3)
			for range 3 {
				entities = append(entities, world.World.NewEntity())
			}

			if variant == 0 {
				// 通常順序
				world.Components.Name.Add(entities[0], &gc.Name{Name: "エンティティA"})
				world.Components.Name.Add(entities[1], &gc.Name{Name: "エンティティB"})
				world.Components.Name.Add(entities[2], &gc.Name{Name: "エンティティC"})
			} else {
				// 逆順
				world.Components.Name.Add(entities[2], &gc.Name{Name: "エンティティC"})
				world.Components.Name.Add(entities[1], &gc.Name{Name: "エンティティB"})
				world.Components.Name.Add(entities[0], &gc.Name{Name: "エンティティA"})
			}

			sm := createTestSerializationManager(t)
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// 両方のバリアントが同じJSONを生成することを確認
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		normalizedJSON := normalizeJSONForComparison(jsonStrings[1])
		assert.JSONEq(t, baseJSON, normalizedJSON,
			"エンティティ作成順序による差異")
	})

	t.Run("プレイヤー生成の決定性確認", func(t *testing.T) {
		t.Parallel()
		jsonStrings := make([]string, 0, 3)

		for range 3 {
			world := testutil.InitTestWorld(t)

			// プレイヤーを生成してリアルなゲームデータを作成
			player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
			require.NoError(t, err)
			professions := raw.PtrSlice(world.Resources.RawMaster.Professions)
			if len(professions) > 0 {
				require.NoError(t, gameaction.ApplyProfession(world, player, professions[0]))
			}

			sm := createTestSerializationManager(t)
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// プレイヤー生成が決定的であることを確認する。
		// 失敗時の差分は assert.JSONEq が表示する
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		for i := 1; i < len(jsonStrings); i++ {
			normalizedJSON := normalizeJSONForComparison(jsonStrings[i])
			assert.JSONEq(t, baseJSON, normalizedJSON,
				"プレイヤー生成セッション %d のJSONが初回と異なります", i+1)
		}
	})

	t.Run("複雑な実世界データの安定性", func(t *testing.T) {
		t.Parallel()
		// 決定的な複雑データを作成
		jsonStrings := make([]string, 0, 3)

		for range 3 {
			world := createComplexDeterministicWorld(t)

			sm := createTestSerializationManager(t)
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// すべてのセッションで同じJSONが生成されることを確認
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		for i := 1; i < len(jsonStrings); i++ {
			normalizedJSON := normalizeJSONForComparison(jsonStrings[i])
			assert.JSONEq(t, baseJSON, normalizedJSON,
				"決定的複雑データセッション %d のJSONが初回と異なります", i+1)
		}
	})
}

// TestSaveLoadRoundTrip セーブ・ロード・再セーブサイクルの包括的テスト
func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("JSON文字列によるラウンドトリップ", func(t *testing.T) {
		t.Parallel()
		// 新しいAPIを使用したメモリ内ラウンドトリップ
		originalWorld := createStandardTestWorld(t)
		sm := createTestSerializationManager(t)

		// JSON生成
		originalJSON, err := sm.GenerateWorldJSON(originalWorld)
		require.NoError(t, err)

		// 新しいワールドに復元
		newWorld := testutil.InitTestWorld(t)

		err = sm.RestoreWorldFromJSON(newWorld, originalJSON)
		require.NoError(t, err)

		// 復元後のワールドから再度JSON生成
		restoredJSON, err := sm.GenerateWorldJSON(newWorld)
		require.NoError(t, err)

		// 正規化して比較
		originalNormalized := normalizeJSONForComparison(originalJSON)
		restoredNormalized := normalizeJSONForComparison(restoredJSON)

		assert.Equal(t, originalNormalized, restoredNormalized,
			"JSON ラウンドトリップで内容が変化しました")
	})

	t.Run("ファイル経由のラウンドトリップ", func(t *testing.T) {
		t.Parallel()
		// 従来のファイル保存APIを使用
		tempDir := t.TempDir()

		sm, smErr := NewSerializationManager(WithSaveDir(tempDir))
		require.NoError(t, smErr)
		originalWorld := createStandardTestWorld(t)

		// 元データ保存
		err := sm.SaveWorld(originalWorld, "original")
		require.NoError(t, err)

		// ロード
		loadedWorld := testutil.InitTestWorld(t)
		err = sm.LoadWorld(loadedWorld, "original")
		require.NoError(t, err)

		// 再保存
		err = sm.SaveWorld(loadedWorld, "reloaded")
		require.NoError(t, err)

		// 内容比較
		originalJSON, err := sm.LoadWorldJSON("original")
		require.NoError(t, err)
		reloadedJSON, err := sm.LoadWorldJSON("reloaded")
		require.NoError(t, err)

		originalNormalized := normalizeJSONForComparison(originalJSON)
		reloadedNormalized := normalizeJSONForComparison(reloadedJSON)

		assert.Equal(t, originalNormalized, reloadedNormalized,
			"ファイル経由ラウンドトリップで内容が変化しました")
	})

	t.Run("多段階ラウンドトリップ", func(t *testing.T) {
		t.Parallel()
		// 複数回のセーブ・ロードサイクル
		tempDir := t.TempDir()

		sm, smErr := NewSerializationManager(WithSaveDir(tempDir))
		require.NoError(t, smErr)
		world := createStandardTestWorld(t)

		contents := make([]string, 0, 3)

		// 複数回のセーブ・ロードサイクル
		for cycle := range 3 {
			filename := "cycle_" + string(rune('0'+cycle))
			err := sm.SaveWorld(world, filename)
			require.NoError(t, err)

			jsonContent, err := sm.LoadWorldJSON(filename)
			require.NoError(t, err)
			contents = append(contents, jsonContent)

			if cycle < 2 {
				// 次のサイクル用にロード
				newWorld := testutil.InitTestWorld(t)
				err = sm.LoadWorld(newWorld, filename)
				require.NoError(t, err)
				world = newWorld
			}
		}

		// すべてのサイクルで同じ内容であることを確認
		baseContent := normalizeJSONForComparison(contents[0])
		for i := 1; i < len(contents); i++ {
			normalizedContent := normalizeJSONForComparison(contents[i])
			assert.Equal(t, baseContent, normalizedContent,
				"サイクル %d で内容が変化しました", i+1)
		}
	})

	t.Run("複雑な実世界データのラウンドトリップ", func(t *testing.T) {
		t.Parallel()
		// 保存対象のみ（プレイヤー、バックパック、装備）のラウンドトリップをテストする
		tempDir := t.TempDir()

		sm, smErr := NewSerializationManager(WithSaveDir(tempDir))
		require.NoError(t, smErr)

		// 保存対象のみを含むワールドを作成
		originalWorld := createStandardTestWorld(t)

		// 元データ保存
		err := sm.SaveWorld(originalWorld, "complex_original")
		require.NoError(t, err)

		// ロード
		loadedWorld := testutil.InitTestWorld(t)
		err = sm.LoadWorld(loadedWorld, "complex_original")
		require.NoError(t, err)

		// 再保存
		err = sm.SaveWorld(loadedWorld, "complex_reloaded")
		require.NoError(t, err)

		// 内容比較
		originalJSON, err := sm.LoadWorldJSON("complex_original")
		require.NoError(t, err)
		reloadedJSON, err := sm.LoadWorldJSON("complex_reloaded")
		require.NoError(t, err)

		originalNormalized := normalizeJSONForComparison(originalJSON)
		reloadedNormalized := normalizeJSONForComparison(reloadedJSON)

		assert.Equal(t, originalNormalized, reloadedNormalized,
			"複雑データラウンドトリップで内容が変化しました")
	})
}

// createStandardTestWorld テスト用の標準的なワールドを作成
func createStandardTestWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)

	// 決定的なエンティティを作成
	player := world.World.NewEntity()
	world.Components.Name.Add(player, &gc.Name{Name: "プレイヤー"})
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.GridElement.Add(player, &gc.GridElement{X: consts.Tile(10), Y: consts.Tile(20)})

	weapon := world.World.NewEntity()
	world.Components.Name.Add(weapon, &gc.Name{Name: "剣"})
	world.Components.Melee.Add(weapon, &gc.Melee{
		Accuracy: 90, Damage: 25, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
	})

	return world
}

// createTestSerializationManager テスト用のSerializationManagerを作成
func createTestSerializationManager(t *testing.T) *SerializationManager {
	t.Helper()
	tempDir := t.TempDir()
	sm, err := NewSerializationManager(WithSaveDir(tempDir))
	require.NoError(t, err)
	return sm
}

// createComplexDeterministicWorld InitNewGameDataのような複雑だが決定的なワールドを作成
func createComplexDeterministicWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)

	// 決定的なプレイヤー作成（手動でコンポーネント追加）
	player := world.World.NewEntity()
	world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.FactionAlly.Add(player, gc.FactionAlly)
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

	// 決定的なアイテム作成（手動でコンポーネント追加）

	// 武器1: 木刀
	sword := world.World.NewEntity()
	world.Components.Name.Add(sword, &gc.Name{Name: "木刀"})
	world.Components.LocationInBackpack.Add(sword, &gc.LocationInBackpack{Owner: player})
	world.Components.Melee.Add(sword, &gc.Melee{
		Accuracy: 100, Damage: 8, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
	})

	// 武器2: ハンドガン
	handgun := world.World.NewEntity()
	world.Components.Name.Add(handgun, &gc.Name{Name: "ハンドガン"})
	world.Components.LocationInBackpack.Add(handgun, &gc.LocationInBackpack{Owner: player})
	world.Components.Melee.Add(handgun, &gc.Melee{
		Accuracy: 85, Damage: 12, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackHandgun,
	})

	// 防具: 西洋鎧
	armor := world.World.NewEntity()
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
	potion := world.World.NewEntity()
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
		Amount: gc.RatioAmount{Ratio: 0.3},
	})

	// 決定的なNPC作成
	for i := range 3 {
		npc := world.World.NewEntity()
		world.Components.Name.Add(npc, &gc.Name{Name: "NPC" + string(rune('A'+i))})
		world.Components.GridElement.Add(npc, &gc.GridElement{
			X: consts.Tile(20 + i*5),
			Y: consts.Tile(25 + i*3),
		})
		world.Components.SoloAI.Add(npc, &gc.SoloAI{ViewDistance: 5})
		world.Components.FactionEnemy.Add(npc, gc.FactionEnemy)
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

	// 決定的なマテリアル追加（手動で作成）
	material1 := world.World.NewEntity()
	world.Components.Name.Add(material1, &gc.Name{Name: "鉄"})
	world.Components.Value.Add(material1, &gc.Value{})
	world.Components.LocationInBackpack.Add(material1, &gc.LocationInBackpack{Owner: player})
	world.Components.Stackable.Add(material1, &gc.Stackable{})

	material2 := world.World.NewEntity()
	world.Components.Name.Add(material2, &gc.Name{Name: "緑ハーブ"})
	world.Components.Value.Add(material2, &gc.Value{})
	world.Components.LocationInBackpack.Add(material2, &gc.LocationInBackpack{Owner: player})
	world.Components.Stackable.Add(material2, &gc.Stackable{})

	return world
}

// normalizeJSONForComparison 比較用にJSONを正規化
func normalizeJSONForComparison(jsonStr string) string {
	lines := make([]string, 0)
	for line := range strings.SplitSeq(jsonStr, "\n") {
		// timestampとchecksumを除外
		if strings.Contains(line, "\"timestamp\"") || strings.Contains(line, "\"checksum\"") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
