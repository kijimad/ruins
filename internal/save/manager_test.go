package save

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestStableIDManager(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	manager := NewStableIDManager()

	// エンティティを作成
	entity1 := world.Manager.NewEntity()
	entity2 := world.Manager.NewEntity()

	// 安定IDを取得
	stableID1 := manager.GetStableID(entity1)
	stableID2 := manager.GetStableID(entity2)

	// IDが異なることを確認
	assert.NotEqual(t, stableID1, stableID2)

	// IDが安定していることを確認
	stableID1Again := manager.GetStableID(entity1)
	assert.Equal(t, stableID1, stableID1Again)

	// 逆引きが正しく動作することを確認
	retrievedEntity1, exists1 := manager.GetEntity(stableID1)
	assert.True(t, exists1)
	assert.Equal(t, entity1, retrievedEntity1)

	retrievedEntity2, exists2 := manager.GetEntity(stableID2)
	assert.True(t, exists2)
	assert.Equal(t, entity2, retrievedEntity2)
}

func TestStableIDGeneration(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	manager := NewStableIDManager()

	// エンティティを作成
	entity1 := world.Manager.NewEntity()
	stableID1 := manager.GetStableID(entity1)

	// エンティティを削除
	manager.UnregisterEntity(entity1)

	// 新しいエンティティを作成
	entity2 := world.Manager.NewEntity()
	stableID2 := manager.GetStableID(entity2)

	// インデックスは再利用されるが、世代が異なることを確認
	if stableID1.Index == stableID2.Index {
		assert.NotEqual(t, stableID1.Generation, stableID2.Generation)
	}

	// 古いIDは無効になることを確認
	assert.False(t, manager.IsValid(stableID1))
	assert.True(t, manager.IsValid(stableID2))
}

func TestComponentRegistry(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	registry := NewComponentRegistry()

	// ワールドから自動初期化
	_ = registry.InitializeFromWorld(world)

	// 型情報が正しく登録されていることを確認（プレイヤー関連コンポーネント）
	nameInfo, exists := registry.GetTypeInfoByName("Name")
	assert.True(t, exists)
	assert.Equal(t, "Name", nameInfo.Name)

	// 存在しない型
	_, exists = registry.GetTypeInfoByName("NonExistent")
	assert.False(t, exists)
}

func TestSerializationManager_SaveAndLoad(t *testing.T) {
	t.Parallel()
	// テストディレクトリを準備
	testDir := "./test_saves"
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// シリアライゼーションマネージャーを作成
	manager := NewSerializationManager(testDir)

	// テスト用ワールドを作成
	world := testutil.InitTestWorld(t)

	// プレイヤーエンティティを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})

	// NPCエンティティを作成（保存されないはず）
	npc := world.Manager.NewEntity()
	npc.AddComponent(world.Components.Name, &gc.Name{Name: "テストNPC"})
	npc.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemyData{})

	// 保存
	err := manager.SaveWorld(world, "test_slot")
	require.NoError(t, err)

	// 新しいワールドを作成
	newWorld := testutil.InitTestWorld(t)

	// 読み込み
	err = manager.LoadWorld(newWorld, "test_slot")
	require.NoError(t, err)

	// プレイヤーが復元されているか確認
	playerCount := 0
	newWorld.Manager.Join(newWorld.Components.Player).Visit(ecs.Visit(func(entity ecs.Entity) {
		playerCount++
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		assert.Equal(t, "テストプレイヤー", name.Name)
	}))

	// NPCが保存されていないことを確認
	npcCount := 0
	newWorld.Manager.Join(newWorld.Components.FactionEnemy).Visit(ecs.Visit(func(_ ecs.Entity) {
		npcCount++
	}))

	assert.Equal(t, 1, playerCount, "プレイヤーが正しくロードされる")
	assert.Equal(t, 0, npcCount, "NPCは保存されない（プレイヤーとアイテムのみ保存）")
}

func TestSerializationManager_EmptyWorld(t *testing.T) {
	t.Parallel()
	// テストディレクトリを準備
	testDir := "./test_saves_empty"
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	// シリアライゼーションマネージャーを作成
	manager := NewSerializationManager(testDir)

	// 空のワールドを作成
	world := testutil.InitTestWorld(t)

	// 保存
	err := manager.SaveWorld(world, "empty_slot")
	require.NoError(t, err)

	// 読み込み
	newWorld := testutil.InitTestWorld(t)
	err = manager.LoadWorld(newWorld, "empty_slot")
	require.NoError(t, err)

	// エンティティが存在しないことを確認
	entityCount := 0
	newWorld.Manager.Join().Visit(ecs.Visit(func(_ ecs.Entity) {
		entityCount++
	}))

	assert.Equal(t, 0, entityCount)
}

func TestValidJSONButNoChecksum(t *testing.T) {
	t.Parallel()
	// テストディレクトリを準備
	testDir := "./test_saves_valid_no_checksum"
	defer func() {
		_ = os.RemoveAll(testDir)
	}()

	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	// 有効なJSONだがチェックサムがないセーブデータを作成
	validJSONNoChecksum := `{
		"version": "1.0.0",
		"timestamp": "2024-01-01T00:00:00Z",
		"world": {
			"entities": []
		}
	}`
	err = os.WriteFile(testDir+"/valid_no_checksum.json", []byte(validJSONNoChecksum), 0644)
	require.NoError(t, err)

	// シリアライゼーションマネージャーを作成
	manager := NewSerializationManager(testDir)
	world := testutil.InitTestWorld(t)

	// 有効なJSONだがチェックサムなしのファイルの読み込み（失敗するはず）
	err = manager.LoadWorld(world, "valid_no_checksum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save data validation failed")
	assert.Contains(t, err.Error(), "checksum field is missing")
}

func TestChecksumValidation(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用のセーブディレクトリ
	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	// テストエンティティを作成
	entity := world.Manager.NewEntity()
	world.Components.Name.Set(entity, &gc.Name{Name: "TestEntity"})

	// セーブ実行
	err := manager.SaveWorld(world, "test_checksum")
	require.NoError(t, err)

	// セーブファイルを読み込み
	data, err := manager.loadDataImpl("test_checksum")
	require.NoError(t, err)

	// JSONをパース
	var saveData Data
	err = json.Unmarshal(data, &saveData)
	require.NoError(t, err)

	// 正常なチェックサム検証
	err = manager.validateChecksum(&saveData)
	assert.NoError(t, err)

	// チェックサムを改ざん
	originalChecksum := saveData.Checksum
	saveData.Checksum = "invalid_checksum"

	// 改ざんされたチェックサムでの検証（失敗するはず）
	err = manager.validateChecksum(&saveData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	// データを改ざん（チェックサムは元に戻す）
	saveData.Checksum = originalChecksum
	saveData.Version = "tampered_version"

	// データ改ざんでの検証（失敗するはず）
	err = manager.validateChecksum(&saveData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestTamperedSaveDataLoad(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用のセーブディレクトリ
	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	// テストエンティティを作成
	entity := world.Manager.NewEntity()
	world.Components.Name.Set(entity, &gc.Name{Name: "TestEntity"})

	// セーブ実行
	err := manager.SaveWorld(world, "test_tampered")
	require.NoError(t, err)

	// セーブファイルを直接読み込み・改ざん
	data, err := manager.loadDataImpl("test_tampered")
	require.NoError(t, err)

	var saveData Data
	err = json.Unmarshal(data, &saveData)
	require.NoError(t, err)

	// データを改ざん
	saveData.Version = "hacked_version"

	// 改ざんされたデータを書き戻し
	tamperedData, err := json.MarshalIndent(saveData, "", "  ")
	require.NoError(t, err)

	err = manager.saveDataImpl("test_tampered", tamperedData)
	require.NoError(t, err)

	// 改ざんされたデータのロードを試行（失敗するはず）
	err = manager.LoadWorld(world, "test_tampered")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save data validation failed")
}

func TestDeterministicHashCalculation(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用のセーブディレクトリ
	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	// 同じテストエンティティを作成
	entity1 := world.Manager.NewEntity()
	world.Components.Name.Set(entity1, &gc.Name{Name: "TestEntity1"})
	entity2 := world.Manager.NewEntity()
	world.Components.Name.Set(entity2, &gc.Name{Name: "TestEntity2"})

	// セーブを複数回実行
	err := manager.SaveWorld(world, "test_deterministic_1")
	require.NoError(t, err)

	err = manager.SaveWorld(world, "test_deterministic_2")
	require.NoError(t, err)

	// 両方のセーブファイルを読み込み
	data1, err := manager.loadDataImpl("test_deterministic_1")
	require.NoError(t, err)

	data2, err := manager.loadDataImpl("test_deterministic_2")
	require.NoError(t, err)

	var saveData1, saveData2 Data
	err = json.Unmarshal(data1, &saveData1)
	require.NoError(t, err)
	err = json.Unmarshal(data2, &saveData2)
	require.NoError(t, err)

	// タイムスタンプを除いてチェックサムを再計算
	// 同じワールド状態であれば、同じチェックサムが生成されるはず
	saveData1.Timestamp = saveData2.Timestamp // タイムスタンプを同一にする

	checksum1 := manager.calculateChecksum(&saveData1)
	checksum2 := manager.calculateChecksum(&saveData2)

	// 決定的ハッシュなので、同じデータからは同じハッシュが生成されるはず
	assert.Equal(t, checksum1, checksum2, "同じワールド状態からは同じチェックサムが生成されるべき")
}

func TestHashConsistencyAcrossRuns(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	// テストエンティティを作成（複数のコンポーネント付き）
	entity := world.Manager.NewEntity()
	world.Components.Name.Set(entity, &gc.Name{Name: "ConsistencyTest"})
	world.Components.Player.Set(entity, &gc.Player{})

	// ワールドデータを抽出
	worldData := manager.extractWorldData(world)

	// 同じデータから複数回ハッシュを計算
	data := Data{
		Version: "1.0.0",
		World:   worldData,
	}

	hash1 := manager.calculateChecksum(&data)
	hash2 := manager.calculateChecksum(&data)
	hash3 := manager.calculateChecksum(&data)

	// 同じデータからは必ず同じハッシュが生成されるはず
	assert.Equal(t, hash1, hash2, "同一データから生成されるハッシュは一致するべき")
	assert.Equal(t, hash2, hash3, "同一データから生成されるハッシュは一致するべき")
	assert.NotEmpty(t, hash1, "ハッシュは空でないべき")
}

func TestMissingChecksumValidation(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	// チェックサムなしのセーブデータを作成
	saveDataWithoutChecksum := Data{
		Version:   "1.0.0",
		Timestamp: time.Now(),
		World: WorldSaveData{
			Entities: []EntitySaveData{},
		},
		// Checksumフィールドは空
	}

	// チェックサム検証（失敗するはず）
	err := manager.validateChecksum(&saveDataWithoutChecksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum field is missing")
}

func TestOldSaveDataWithoutChecksum(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	// テストエンティティを作成
	entity := world.Manager.NewEntity()
	world.Components.Name.Set(entity, &gc.Name{Name: "TestEntity"})

	// チェックサムなしの古いフォーマットのセーブデータを手動作成
	oldFormatData := map[string]interface{}{
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
		"world": map[string]interface{}{
			"entities": []interface{}{},
		},
		// checksumフィールドなし
	}

	// JSONにシリアライズ
	oldFormatJSON, err := json.MarshalIndent(oldFormatData, "", "  ")
	require.NoError(t, err)

	// セーブファイルとして書き込み
	err = manager.saveDataImpl("old_format_test", oldFormatJSON)
	require.NoError(t, err)

	// 古いフォーマットのロードを試行（失敗するはず）
	err = manager.LoadWorld(world, "old_format_test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save data validation failed")
	assert.Contains(t, err.Error(), "checksum field is missing")
}
