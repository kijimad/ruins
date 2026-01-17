package save

import (
	"os"
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TestJSONDeterministicBehavior JSON出力の決定的動作を包括的にテスト
func TestJSONDeterministicBehavior(t *testing.T) {
	t.Parallel()

	t.Run("同一セッション内での安定性", func(t *testing.T) {
		t.Parallel()
		// 同じワールドインスタンスで複数回JSON生成
		world := createStandardTestWorld(t)
		sm := createTestSerializationManager(t)

		var jsonStrings []string
		for i := 0; i < 5; i++ {
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// すべて同一であることを確認
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		for i := 1; i < len(jsonStrings); i++ {
			normalizedJSON := normalizeJSONForComparison(jsonStrings[i])
			assert.Equal(t, baseJSON, normalizedJSON,
				"同じワールドから生成されたJSON %d が一致しません", i+1)
		}
	})

	t.Run("異なるセッション間での安定性", func(t *testing.T) {
		t.Parallel()
		// 異なるワールドインスタンスで同じデータを作成
		var jsonStrings []string
		for session := 0; session < 3; session++ {
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
			assert.Equal(t, baseJSON, normalizedJSON,
				"セッション %d のJSONが一致しません", i+1)
		}
	})

	t.Run("コンポーネント追加順序に依存しない", func(t *testing.T) {
		t.Parallel()
		// 異なる順序でコンポーネントを追加したワールドを作成
		var jsonStrings []string

		for variant := 0; variant < 3; variant++ {
			world := testutil.InitTestWorld(t)

			entity := world.Manager.NewEntity()

			// バリアント毎に異なる順序でコンポーネントを追加
			switch variant {
			case 0:
				// 順序: Name -> GridElement -> Attack
				entity.AddComponent(world.Components.Name, &gc.Name{Name: "テストエンティティ"})
				entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: gc.Tile(1), Y: gc.Tile(1)})
				entity.AddComponent(world.Components.Attack, &gc.Attack{
					Accuracy: 85, Damage: 20, AttackCount: 1,
					Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
				})
			case 1:
				// 順序: Attack -> Name -> GridElement
				entity.AddComponent(world.Components.Attack, &gc.Attack{
					Accuracy: 85, Damage: 20, AttackCount: 1,
					Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
				})
				entity.AddComponent(world.Components.Name, &gc.Name{Name: "テストエンティティ"})
				entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: gc.Tile(1), Y: gc.Tile(1)})
			case 2:
				// 順序: GridElement -> Attack -> Name
				entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: gc.Tile(1), Y: gc.Tile(1)})
				entity.AddComponent(world.Components.Attack, &gc.Attack{
					Accuracy: 85, Damage: 20, AttackCount: 1,
					Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
				})
				entity.AddComponent(world.Components.Name, &gc.Name{Name: "テストエンティティ"})
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
			assert.Equal(t, baseJSON, normalizedJSON,
				"コンポーネント追加順序による差異 (variant %d)", i+1)
		}
	})

	t.Run("エンティティ作成順序に依存しない", func(t *testing.T) {
		t.Parallel()
		// 異なる順序でエンティティを作成
		var jsonStrings []string

		for variant := 0; variant < 2; variant++ {
			world := testutil.InitTestWorld(t)

			var entities []ecs.Entity
			for i := 0; i < 3; i++ {
				entities = append(entities, world.Manager.NewEntity())
			}

			if variant == 0 {
				// 通常順序
				entities[0].AddComponent(world.Components.Name, &gc.Name{Name: "エンティティA"})
				entities[1].AddComponent(world.Components.Name, &gc.Name{Name: "エンティティB"})
				entities[2].AddComponent(world.Components.Name, &gc.Name{Name: "エンティティC"})
			} else {
				// 逆順
				entities[2].AddComponent(world.Components.Name, &gc.Name{Name: "エンティティC"})
				entities[1].AddComponent(world.Components.Name, &gc.Name{Name: "エンティティB"})
				entities[0].AddComponent(world.Components.Name, &gc.Name{Name: "エンティティA"})
			}

			sm := createTestSerializationManager(t)
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// 両方のバリアントが同じJSONを生成することを確認
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		normalizedJSON := normalizeJSONForComparison(jsonStrings[1])
		assert.Equal(t, baseJSON, normalizedJSON,
			"エンティティ作成順序による差異")
	})

	t.Run("InitDebugDataの決定性確認", func(t *testing.T) {
		t.Parallel()
		var jsonStrings []string

		for session := 0; session < 3; session++ {
			world := testutil.InitTestWorld(t)

			// InitDebugDataを使用してリアルなゲームデータを作成
			worldhelper.InitDebugData(world)

			sm := createTestSerializationManager(t)
			jsonStr, err := sm.GenerateWorldJSON(world)
			require.NoError(t, err)
			jsonStrings = append(jsonStrings, jsonStr)
		}

		// InitDebugDataが決定的であることを確認
		baseJSON := normalizeJSONForComparison(jsonStrings[0])
		for i := 1; i < len(jsonStrings); i++ {
			normalizedJSON := normalizeJSONForComparison(jsonStrings[i])

			if baseJSON != normalizedJSON {
				t.Errorf("InitDebugDataセッション %d のJSONが一致しません（修正後も非決定的）", i+1)

				// 差分の詳細を表示
				baseLines := strings.Split(baseJSON, "\n")
				compareLines := strings.Split(normalizedJSON, "\n")

				diffCount := 0
				maxDiffs := 10
				for lineNum := 0; lineNum < len(baseLines) && lineNum < len(compareLines) && diffCount < maxDiffs; lineNum++ {
					if baseLines[lineNum] != compareLines[lineNum] {
						t.Logf("行 %d の違い:", lineNum+1)
						t.Logf("  セッション1: %s", baseLines[lineNum])
						t.Logf("  セッション%d: %s", i+1, compareLines[lineNum])
						diffCount++
					}
				}
				if diffCount >= maxDiffs {
					t.Logf("... (さらに %d個以上の違いが見つかりました)", maxDiffs)
				}
				break
			}
		}

		// すべてのセッションで同一のJSONが生成されることを確認
		for i := 1; i < len(jsonStrings); i++ {
			normalizedJSON := normalizeJSONForComparison(jsonStrings[i])
			assert.Equal(t, baseJSON, normalizedJSON,
				"InitDebugDataセッション %d のJSONが初回と異なります", i+1)
		}
	})

	t.Run("複雑な実世界データの安定性", func(t *testing.T) {
		t.Parallel()
		// 決定的な複雑データを作成
		var jsonStrings []string

		for session := 0; session < 3; session++ {
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
			assert.Equal(t, baseJSON, normalizedJSON,
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
		tempDir, err := os.MkdirTemp("", "round_trip_test_")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		sm := NewSerializationManager(tempDir)
		originalWorld := createStandardTestWorld(t)

		// 元データ保存
		err = sm.SaveWorld(originalWorld, "original")
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
		tempDir, err := os.MkdirTemp("", "multi_round_trip_test_")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		sm := NewSerializationManager(tempDir)
		world := createStandardTestWorld(t)

		var contents []string

		// 複数回のセーブ・ロードサイクル
		for cycle := 0; cycle < 3; cycle++ {
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
		tempDir, err := os.MkdirTemp("", "complex_round_trip_test_")
		require.NoError(t, err)
		defer func() {
			_ = os.RemoveAll(tempDir)
		}()

		sm := NewSerializationManager(tempDir)

		// 保存対象のみを含むワールドを作成
		originalWorld := createStandardTestWorld(t)

		// 元データ保存
		err = sm.SaveWorld(originalWorld, "complex_original")
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
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Name, &gc.Name{Name: "プレイヤー"})
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: gc.Tile(10), Y: gc.Tile(20)})

	weapon := world.Manager.NewEntity()
	weapon.AddComponent(world.Components.Name, &gc.Name{Name: "剣"})
	weapon.AddComponent(world.Components.Item, &gc.Item{})
	weapon.AddComponent(world.Components.Attack, &gc.Attack{
		Accuracy: 90, Damage: 25, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
	})

	return world
}

// createTestSerializationManager テスト用のSerializationManagerを作成
func createTestSerializationManager(t *testing.T) *SerializationManager {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "test_sm_")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})
	return NewSerializationManager(tempDir)
}

// createComplexDeterministicWorld InitDebugDataのような複雑だが決定的なワールドを作成
func createComplexDeterministicWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)

	// 決定的なプレイヤー作成（手動でコンポーネント追加）
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.FactionAlly, gc.FactionAlly)
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: gc.Tile(10), Y: gc.Tile(15)})
	player.AddComponent(world.Components.Attributes, &gc.Attributes{
		Vitality:  gc.Attribute{Base: 10, Modifier: 0, Total: 10},
		Strength:  gc.Attribute{Base: 8, Modifier: 0, Total: 8},
		Sensation: gc.Attribute{Base: 6, Modifier: 0, Total: 6},
		Dexterity: gc.Attribute{Base: 7, Modifier: 0, Total: 7},
		Agility:   gc.Attribute{Base: 9, Modifier: 0, Total: 9},
		Defense:   gc.Attribute{Base: 5, Modifier: 0, Total: 5},
	})
	player.AddComponent(world.Components.Pools, &gc.Pools{
		HP: gc.Pool{Current: 100, Max: 100},
		SP: gc.Pool{Current: 50, Max: 50},
		EP: gc.Pool{Current: 30, Max: 30},
	})

	// 決定的なアイテム作成（手動でコンポーネント追加）
	var items []ecs.Entity

	// 武器1: 木刀
	sword := world.Manager.NewEntity()
	sword.AddComponent(world.Components.Name, &gc.Name{Name: "木刀"})
	sword.AddComponent(world.Components.Item, &gc.Item{})
	sword.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
	sword.AddComponent(world.Components.Attack, &gc.Attack{
		Accuracy: 100, Damage: 8, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackSword,
	})
	_ = append(items, sword)

	// 武器2: ハンドガン
	handgun := world.Manager.NewEntity()
	handgun.AddComponent(world.Components.Name, &gc.Name{Name: "ハンドガン"})
	handgun.AddComponent(world.Components.Item, &gc.Item{})
	handgun.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
	handgun.AddComponent(world.Components.Attack, &gc.Attack{
		Accuracy: 85, Damage: 12, AttackCount: 1,
		Element: gc.ElementTypeNone, AttackCategory: gc.AttackHandgun,
	})
	_ = append(items, handgun)

	// 防具: 西洋鎧
	armor := world.Manager.NewEntity()
	armor.AddComponent(world.Components.Name, &gc.Name{Name: "西洋鎧"})
	armor.AddComponent(world.Components.Item, &gc.Item{})
	armor.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
	armor.AddComponent(world.Components.Wearable, &gc.Wearable{
		Defense:           15,
		EquipmentCategory: gc.EquipmentTorso,
		EquipBonus: gc.EquipBonus{
			Vitality: 2, Strength: 1, Sensation: 0, Dexterity: 0, Agility: -1,
		},
	})
	_ = append(items, armor)

	// 回復アイテム
	potion := world.Manager.NewEntity()
	potion.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})
	potion.AddComponent(world.Components.Item, &gc.Item{})
	potion.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
	potion.AddComponent(world.Components.Consumable, &gc.Consumable{
		UsableScene: gc.UsableSceneAny,
		TargetType: gc.TargetType{
			TargetGroup: gc.TargetGroupAlly,
			TargetNum:   gc.TargetSingle,
		},
	})
	potion.AddComponent(world.Components.ProvidesHealing, &gc.ProvidesHealing{
		Amount: gc.RatioAmount{Ratio: 0.3},
	})
	_ = append(items, potion)

	// 決定的なNPC作成
	for i := 0; i < 3; i++ {
		npc := world.Manager.NewEntity()
		npc.AddComponent(world.Components.Name, &gc.Name{Name: "NPC" + string(rune('A'+i))})
		npc.AddComponent(world.Components.GridElement, &gc.GridElement{
			X: gc.Tile(20 + i*5),
			Y: gc.Tile(25 + i*3),
		})
		npc.AddComponent(world.Components.AIVision, &gc.AIVision{ViewDistance: gc.Pixel(160)})
		npc.AddComponent(world.Components.FactionEnemy, gc.FactionEnemy)
		npc.AddComponent(world.Components.Attributes, &gc.Attributes{
			Vitality:  gc.Attribute{Base: 10 + i, Modifier: 0, Total: 10 + i},
			Strength:  gc.Attribute{Base: 8 + i, Modifier: 0, Total: 8 + i},
			Sensation: gc.Attribute{Base: 6 + i, Modifier: 0, Total: 6 + i},
			Dexterity: gc.Attribute{Base: 7 + i, Modifier: 0, Total: 7 + i},
			Agility:   gc.Attribute{Base: 9 + i, Modifier: 0, Total: 9 + i},
			Defense:   gc.Attribute{Base: 5 + i, Modifier: 0, Total: 5 + i},
		})
		npc.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 100 + i*10, Max: 100 + i*10},
			SP: gc.Pool{Current: 50 + i*5, Max: 50 + i*5},
			EP: gc.Pool{Current: 30 + i*3, Max: 30 + i*3},
		})
	}

	// 決定的なマテリアル追加（手動で作成）
	material1 := world.Manager.NewEntity()
	material1.AddComponent(world.Components.Name, &gc.Name{Name: "鉄"})
	material1.AddComponent(world.Components.Item, &gc.Item{Count: 40})
	material1.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
	material1.AddComponent(world.Components.Stackable, &gc.Stackable{})

	material2 := world.Manager.NewEntity()
	material2.AddComponent(world.Components.Name, &gc.Name{Name: "緑ハーブ"})
	material2.AddComponent(world.Components.Item, &gc.Item{Count: 2})
	material2.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
	material2.AddComponent(world.Components.Stackable, &gc.Stackable{})

	return world
}

// normalizeJSONForComparison 比較用にJSONを正規化
func normalizeJSONForComparison(jsonStr string) string {
	lines := make([]string, 0)
	for _, line := range strings.Split(jsonStr, "\n") {
		// timestampとchecksumを除外
		if strings.Contains(line, "\"timestamp\"") || strings.Contains(line, "\"checksum\"") {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
