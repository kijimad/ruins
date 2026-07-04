package save

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestStableIDManager(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	manager := NewStableIDManager()

	entity1 := world.Manager.NewEntity()
	entity2 := world.Manager.NewEntity()

	stableID1 := manager.GetStableID(entity1)
	stableID2 := manager.GetStableID(entity2)

	assert.NotEqual(t, stableID1, stableID2)

	stableID1Again := manager.GetStableID(entity1)
	assert.Equal(t, stableID1, stableID1Again)

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

	entity1 := world.Manager.NewEntity()
	stableID1 := manager.GetStableID(entity1)

	manager.UnregisterEntity(entity1)

	entity2 := world.Manager.NewEntity()
	stableID2 := manager.GetStableID(entity2)

	if stableID1.Index == stableID2.Index {
		assert.NotEqual(t, stableID1.Generation, stableID2.Generation)
	}

	assert.False(t, manager.IsValid(stableID1))
	assert.True(t, manager.IsValid(stableID2))
}

func TestSerializationManager_SaveAndLoad(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	manager := NewSerializationManager(testDir)
	world := testutil.InitTestWorld(t)

	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.Name, &gc.Name{Name: "テストプレイヤー"})

	npc := world.Manager.NewEntity()
	npc.AddComponent(world.Components.Name, &gc.Name{Name: "テストNPC"})
	npc.AddComponent(world.Components.FactionEnemy, &gc.FactionEnemyData{})

	err := manager.SaveWorld(world, "test_slot")
	require.NoError(t, err)

	newWorld := testutil.InitTestWorld(t)
	err = manager.LoadWorld(newWorld, "test_slot")
	require.NoError(t, err)

	playerCount := 0
	newWorld.Manager.Join(newWorld.Components.Player).Visit(ecs.Visit(func(entity ecs.Entity) {
		playerCount++
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		assert.Equal(t, "テストプレイヤー", name.Name)
	}))

	npcCount := 0
	newWorld.Manager.Join(newWorld.Components.FactionEnemy).Visit(ecs.Visit(func(_ ecs.Entity) {
		npcCount++
	}))

	assert.Equal(t, 1, playerCount, "プレイヤーが正しくロードされる")
	assert.Equal(t, 0, npcCount, "NPCは保存されない（プレイヤーとアイテムのみ保存）")
}

func TestSerializationManager_EmptyWorld(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	manager := NewSerializationManager(testDir)
	world := testutil.InitTestWorld(t)

	err := manager.SaveWorld(world, "empty_slot")
	require.NoError(t, err)

	newWorld := testutil.InitTestWorld(t)
	err = manager.LoadWorld(newWorld, "empty_slot")
	require.NoError(t, err)

	entityCount := 0
	singleton := newWorld.Resources.SingletonEntity
	newWorld.Manager.Join().Visit(ecs.Visit(func(entity ecs.Entity) {
		if entity != singleton {
			entityCount++
		}
	}))

	assert.Equal(t, 0, entityCount)
}

func TestValidJSONButNoChecksum(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	validJSONNoChecksum := `{
		"version": "1.0.0",
		"timestamp": "2024-01-01T00:00:00Z",
		"world": {
			"entities": []
		}
	}`
	err := os.WriteFile(testDir+"/valid_no_checksum.json", []byte(validJSONNoChecksum), 0644)
	require.NoError(t, err)

	manager := NewSerializationManager(testDir)
	world := testutil.InitTestWorld(t)

	err = manager.LoadWorld(world, "valid_no_checksum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "バリデーションエラー")
	assert.Contains(t, err.Error(), "checksum")
}

func TestChecksumValidation(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	entity := world.Manager.NewEntity()
	world.Components.Name.Set(entity, &gc.Name{Name: "TestEntity"})

	err := manager.SaveWorld(world, "test_checksum")
	require.NoError(t, err)

	data, err := manager.loadDataImpl("test_checksum")
	require.NoError(t, err)

	var saveData oapi.SaveDataSaveData
	err = json.Unmarshal(data, &saveData)
	require.NoError(t, err)

	// 正常なチェックサム検証
	err = manager.validateChecksum(&saveData)
	assert.NoError(t, err)

	// チェックサムを改ざん
	originalChecksum := saveData.Checksum
	saveData.Checksum = "invalid_checksum"
	err = manager.validateChecksum(&saveData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	// データを改ざん（チェックサムは元に戻す）
	saveData.Checksum = originalChecksum
	saveData.Version = "tampered_version"
	err = manager.validateChecksum(&saveData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestTamperedSaveDataLoad(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	entity := world.Manager.NewEntity()
	world.Components.Name.Set(entity, &gc.Name{Name: "TestEntity"})

	err := manager.SaveWorld(world, "test_tampered")
	require.NoError(t, err)

	data, err := manager.loadDataImpl("test_tampered")
	require.NoError(t, err)

	var saveData oapi.SaveDataSaveData
	err = json.Unmarshal(data, &saveData)
	require.NoError(t, err)

	saveData.Version = "hacked_version"

	tamperedData, err := json.MarshalIndent(saveData, "", "  ")
	require.NoError(t, err)

	err = manager.saveDataImpl("test_tampered", tamperedData)
	require.NoError(t, err)

	err = manager.LoadWorld(world, "test_tampered")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "バリデーションエラー")
}

func TestDeterministicHashCalculation(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	entity1 := world.Manager.NewEntity()
	world.Components.Name.Set(entity1, &gc.Name{Name: "TestEntity1"})
	entity2 := world.Manager.NewEntity()
	world.Components.Name.Set(entity2, &gc.Name{Name: "TestEntity2"})

	err := manager.SaveWorld(world, "test_deterministic_1")
	require.NoError(t, err)

	err = manager.SaveWorld(world, "test_deterministic_2")
	require.NoError(t, err)

	data1, err := manager.loadDataImpl("test_deterministic_1")
	require.NoError(t, err)

	data2, err := manager.loadDataImpl("test_deterministic_2")
	require.NoError(t, err)

	var saveData1, saveData2 oapi.SaveDataSaveData
	err = json.Unmarshal(data1, &saveData1)
	require.NoError(t, err)
	err = json.Unmarshal(data2, &saveData2)
	require.NoError(t, err)

	// タイムスタンプを同一にしてチェックサムを再計算
	saveData1.Timestamp = saveData2.Timestamp

	checksum1, err := manager.calculateChecksum(&saveData1)
	require.NoError(t, err)
	checksum2, err := manager.calculateChecksum(&saveData2)
	require.NoError(t, err)

	assert.Equal(t, checksum1, checksum2, "同じワールド状態からは同じチェックサムが生成されるべき")
}

func TestHashConsistencyAcrossRuns(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	entity := world.Manager.NewEntity()
	world.Components.Name.Set(entity, &gc.Name{Name: "ConsistencyTest"})
	world.Components.Player.Set(entity, &gc.Player{})

	worldData := manager.extractWorldData(world)

	saveData := oapi.SaveDataSaveData{
		Version: "1.0.0",
		World:   worldData,
	}

	hash1, err := manager.calculateChecksum(&saveData)
	require.NoError(t, err)
	hash2, err := manager.calculateChecksum(&saveData)
	require.NoError(t, err)
	hash3, err := manager.calculateChecksum(&saveData)
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2, "同一データから生成されるハッシュは一致するべき")
	assert.Equal(t, hash2, hash3, "同一データから生成されるハッシュは一致するべき")
	assert.NotEmpty(t, hash1, "ハッシュは空でないべき")
}

func TestMissingChecksumValidation(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	saveDataWithoutChecksum := oapi.SaveDataSaveData{
		Version:   "1.0.0",
		Timestamp: time.Now(),
		World: oapi.SaveDataWorldSaveData{
			Entities: []oapi.SaveDataEntitySaveData{},
		},
	}

	err := manager.validateChecksum(&saveDataWithoutChecksum)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum field is missing")
}

func TestOldSaveDataWithoutChecksum(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	tempDir := t.TempDir()
	manager := NewSerializationManager(tempDir)

	entity := world.Manager.NewEntity()
	world.Components.Name.Set(entity, &gc.Name{Name: "TestEntity"})

	oldFormatData := map[string]any{
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
		"world": map[string]any{
			"entities": []any{},
		},
	}

	oldFormatJSON, err := json.MarshalIndent(oldFormatData, "", "  ")
	require.NoError(t, err)

	err = manager.saveDataImpl("old_format_test", oldFormatJSON)
	require.NoError(t, err)

	err = manager.LoadWorld(world, "old_format_test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "バリデーションエラー")
	assert.Contains(t, err.Error(), "checksum")
}
