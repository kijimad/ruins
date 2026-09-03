package save

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializationManager_SaveAndLoad(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)
	world := testutil.InitTestWorld(t)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.Name.Add(player, &gc.Name{Name: "テストプレイヤー"})

	npc := world.ECS.NewEntity()
	world.Components.Name.Add(npc, &gc.Name{Name: "テストNPC"})
	world.Components.FactionEnemy.Add(npc, &gc.FactionEnemy{})

	err = manager.SaveWorld(world, "test_slot")
	require.NoError(t, err)

	newWorld := testutil.InitTestWorld(t)
	err = manager.LoadWorld(newWorld, "test_slot")
	require.NoError(t, err)

	playerCount := 0
	playerQuery := ecs.NewFilter1[gc.Player](newWorld.ECS).Query()
	for playerQuery.Next() {
		entity := playerQuery.Entity()
		playerCount++
		name := newWorld.Components.Name.Get(entity)
		assert.Equal(t, "テストプレイヤー", name.Name)
	}

	npcCount := 0
	npcQuery := ecs.NewFilter1[gc.FactionEnemy](newWorld.ECS).Query()
	for npcQuery.Next() {
		npcCount++
	}

	assert.Equal(t, 1, playerCount, "プレイヤーが正しくロードされる")
	assert.Equal(t, 1, npcCount, "丸ごと保存のためNPCも保存・復元される")
}

func TestSerializationManager_EmptyWorld(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)
	world := testutil.InitTestWorld(t)

	err = manager.SaveWorld(world, "empty_slot")
	require.NoError(t, err)

	newWorld := testutil.InitTestWorld(t)
	err = manager.LoadWorld(newWorld, "empty_slot")
	require.NoError(t, err)

	// 空スロットをロードしても、基盤エンティティ以外は湧かないことを確かめる。
	// 丸ごと保存方式でも、ゲーム内容がゼロなら復元後もゼロという不変条件。
	//
	// 基盤エンティティは2種。全シングルトン成分を載せた singleton が1個と、現ステージの寸法を
	// 持つ StageField が1個。どちらも world 初期化とロードが必ず作るので、数えずに除外する。
	// Filter0 は成分を問わず全エンティティを反復する。除外して残るのがゲーム内容で、空セーブなら0。
	gameEntityCount := 0
	singleton := newWorld.Resources.SingletonEntity
	allEntities := ecs.NewFilter0(newWorld.ECS).Query()
	for allEntities.Next() {
		entity := allEntities.Entity()
		isInfra := entity == singleton || newWorld.Components.StageField.Has(entity)
		if !isInfra {
			gameEntityCount++
		}
	}

	assert.Equal(t, 0, gameEntityCount, "空セーブのロードでゲーム内容は湧かない")
}

func TestValidJSONButNoChecksum(t *testing.T) {
	t.Parallel()
	testDir := t.TempDir()

	validJSONNoChecksum := `{
		"version": "1.0.0",
		"timestamp": "2024-01-01T00:00:00Z",
		"world": {}
	}`

	manager, err := NewSerializationManager(WithSaveDir(testDir))
	require.NoError(t, err)
	require.NoError(t, manager.storeSaveJSON("valid_no_checksum", []byte(validJSONNoChecksum)))

	world := testutil.InitTestWorld(t)

	err = manager.LoadWorld(world, "valid_no_checksum")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum")
}

func TestChecksumValidation(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	tempDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(tempDir))
	require.NoError(t, err)

	entity := world.ECS.NewEntity()
	world.Components.Name.Add(entity, &gc.Name{Name: "TestEntity"})

	err = manager.SaveWorld(world, "test_checksum")
	require.NoError(t, err)

	data, err := manager.loadSaveJSON("test_checksum")
	require.NoError(t, err)

	var env saveEnvelope
	err = json.Unmarshal(data, &env)
	require.NoError(t, err)

	// 正常なチェックサム検証
	err = validateChecksum(&env)
	require.NoError(t, err)

	// チェックサムを改ざん
	originalChecksum := env.Checksum
	env.Checksum = "invalid_checksum"
	err = validateChecksum(&env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")

	// データを改ざん（チェックサムは元に戻す）
	env.Checksum = originalChecksum
	env.Version = "tampered_version"
	err = validateChecksum(&env)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestTamperedSaveDataLoad(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	tempDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(tempDir))
	require.NoError(t, err)

	entity := world.ECS.NewEntity()
	world.Components.Name.Add(entity, &gc.Name{Name: "TestEntity"})

	err = manager.SaveWorld(world, "test_tampered")
	require.NoError(t, err)

	data, err := manager.loadSaveJSON("test_tampered")
	require.NoError(t, err)

	var env saveEnvelope
	err = json.Unmarshal(data, &env)
	require.NoError(t, err)

	// バージョンを改ざんするとチェックサムが合わなくなる
	env.Version = "hacked_version"

	tamperedData, err := json.MarshalIndent(env, "", "  ")
	require.NoError(t, err)

	err = manager.storeSaveJSON("test_tampered", tamperedData)
	require.NoError(t, err)

	err = manager.LoadWorld(world, "test_tampered")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestHashConsistencyAcrossRuns(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	entity := world.ECS.NewEntity()
	world.Components.Name.Add(entity, &gc.Name{Name: "ConsistencyTest"})
	world.Components.Player.Add(entity, &gc.Player{})

	worldJSON, err := serializeWorld(world)
	require.NoError(t, err)

	env := saveEnvelope{
		Version: "1.0.0",
		World:   worldJSON,
	}

	hash1 := checksumOf(&env)
	hash2 := checksumOf(&env)
	hash3 := checksumOf(&env)

	assert.Equal(t, hash1, hash2, "同一データから生成されるハッシュは一致するべき")
	assert.Equal(t, hash2, hash3, "同一データから生成されるハッシュは一致するべき")
	assert.NotEmpty(t, hash1, "ハッシュは空でないべき")
}

func TestMissingChecksumValidation(t *testing.T) {
	t.Parallel()

	err := validateChecksum(&saveEnvelope{
		Version:   "1.0.0",
		Timestamp: time.Now(),
		World:     json.RawMessage(`{}`),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum field is missing")
}

func TestOldSaveDataWithoutChecksum(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	tempDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(tempDir))
	require.NoError(t, err)

	entity := world.ECS.NewEntity()
	world.Components.Name.Add(entity, &gc.Name{Name: "TestEntity"})

	oldFormatData := map[string]any{
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
		"world":     map[string]any{},
	}

	oldFormatJSON, err := json.MarshalIndent(oldFormatData, "", "  ")
	require.NoError(t, err)

	err = manager.storeSaveJSON("old_format_test", oldFormatJSON)
	require.NoError(t, err)

	err = manager.LoadWorld(world, "old_format_test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum")
}

func TestListSaves(t *testing.T) {
	t.Parallel()

	t.Run("空の場合は空リストを返す", func(t *testing.T) {
		t.Parallel()
		emptyDir := t.TempDir()
		m, err := NewSerializationManager(WithSaveDir(emptyDir))
		require.NoError(t, err)
		saves, err := m.ListSaves()
		require.NoError(t, err)
		assert.Empty(t, saves)
	})

	t.Run("手動セーブとオートセーブを区別して一覧する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.ECS.NewEntity()
		world.Components.Name.Add(entity, &gc.Name{Name: "Ash"})
		world.Components.Player.Add(entity, &gc.Player{})

		dir := t.TempDir()
		m, err := NewSerializationManager(WithSaveDir(dir))
		require.NoError(t, err)
		require.NoError(t, m.SaveWorld(world, "slot1"))
		require.NoError(t, m.SaveWorld(world, "auto_20260704_1830"))

		saves, err := m.ListSaves()
		require.NoError(t, err)
		assert.Len(t, saves, 2)

		autoCount := 0
		manualCount := 0
		for _, name := range saves {
			if strings.HasPrefix(name, autoSavePrefix) {
				autoCount++
			} else {
				manualCount++
			}
		}
		assert.Equal(t, 1, autoCount)
		assert.Equal(t, 1, manualCount)
	})
}

func TestAutoSaveRotation(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	tempDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(tempDir))
	require.NoError(t, err)

	entity := world.ECS.NewEntity()
	world.Components.Name.Add(entity, &gc.Name{Name: "Ash"})
	world.Components.Player.Add(entity, &gc.Player{})

	// 先に2件作る
	for range 2 {
		require.NoError(t, manager.AutoSave(world))
	}
	earlySaves, err := manager.ListAutoSaves()
	require.NoError(t, err)

	// さらに maxAutoSaves 件作ってローテーションを発動させる
	for range maxAutoSaves {
		require.NoError(t, manager.AutoSave(world))
	}

	autoSaves, err := manager.ListAutoSaves()
	require.NoError(t, err)
	assert.Len(t, autoSaves, maxAutoSaves)

	// 古い2件は削除されている
	for _, name := range earlySaves {
		assert.False(t, manager.SaveFileExists(name), "古いオートセーブ %s は削除されている", name)
	}
}

func TestGetSavePlayerName(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	tempDir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(tempDir))
	require.NoError(t, err)

	entity := world.ECS.NewEntity()
	world.Components.Name.Add(entity, &gc.Name{Name: "Ash"})
	world.Components.Player.Add(entity, &gc.Player{})

	require.NoError(t, manager.SaveWorld(world, "slot1"))

	name, err := manager.GetSavePlayerName("slot1")
	require.NoError(t, err)
	assert.Equal(t, "Ash", name)

	_, err = manager.GetSavePlayerName("nonexistent")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

// TestRestoreWorldFromJSON_VersionMismatch はサポート外バージョンのセーブを拒否することを検証する。
// バージョン検証はチェックサム検証の後に行われるため、改ざん後に正しいチェックサムを再計算する。
func TestRestoreWorldFromJSON_VersionMismatch(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	jsonData, err := manager.GenerateWorldJSON(world)
	require.NoError(t, err)

	var env saveEnvelope
	require.NoError(t, json.Unmarshal([]byte(jsonData), &env))
	env.Version = "0.0.0-unsupported"
	env.Checksum = checksumOf(&env)
	tampered, err := json.Marshal(env)
	require.NoError(t, err)

	newWorld := testutil.InitTestWorld(t)
	err = manager.RestoreWorldFromJSON(newWorld, string(tampered))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported save data version")
}

// TestRestoreWorldFromJSON_InvalidJSON は壊れたJSONを拒否することを検証する。
func TestRestoreWorldFromJSON_InvalidJSON(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	err = manager.RestoreWorldFromJSON(world, "{ 壊れたJSON ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

// TestRestoreWorldFromJSON_MissingSingleton は復元データにシングルトン（Dungeon保持
// エンティティ）が無い場合にエラーを返すことを検証する。
func TestRestoreWorldFromJSON_MissingSingleton(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	// シングルトンからDungeonを取り除くと、復元時にシングルトンを特定できない
	world.Components.Dungeon.Remove(world.Resources.SingletonEntity)
	require.NoError(t, manager.SaveWorld(world, "no_singleton"))

	newWorld := testutil.InitTestWorld(t)
	err = manager.LoadWorld(newWorld, "no_singleton")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "singleton")
}

// TestNewSerializationManager_保存ディレクトリの作成に失敗するとエラー は、セーブディレクトリの
// 親パス上に同名ファイルが既にあり MkdirAll が失敗するケースを検証する。
func TestNewSerializationManager_保存ディレクトリの作成に失敗するとエラー(t *testing.T) {
	t.Parallel()

	// blocker をファイルとして作っておくと、その配下をディレクトリとして MkdirAll できない
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))

	_, err := NewSerializationManager(WithSaveDir(filepath.Join(blocker, "saves")))
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to create save directory")
}

// TestSerializationManager_ListSaves_保存ディレクトリの読み込みに失敗するとエラー は、
// 生成後にセーブディレクトリ自体が失われた場合に ListSaves がエラーを返すことを検証する。
func TestSerializationManager_ListSaves_保存ディレクトリの読み込みに失敗するとエラー(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(dir))
	require.NoError(t, err)

	// この時点でディレクトリは空なので os.Remove で削除できる
	require.NoError(t, os.Remove(dir))

	_, err = manager.ListSaves()
	require.ErrorIs(t, err, fs.ErrNotExist)
}

// TestSerializationManager_deleteSaveImpl_存在しないスロットの削除はエラー は、
// 実ファイルが無いスロット名を消そうとしたときに os.Remove 由来のエラーが伝播することを検証する。
func TestSerializationManager_deleteSaveImpl_存在しないスロットの削除はエラー(t *testing.T) {
	t.Parallel()

	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	err = manager.deleteSaveImpl("does_not_exist")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

// TestSerializationManager_LoadWorldJSON_存在しないスロットはエラー は、
// LoadWorldJSON が読み込み失敗をラップしつつ、元のファイル未検出エラーを保持することを検証する。
func TestSerializationManager_LoadWorldJSON_存在しないスロットはエラー(t *testing.T) {
	t.Parallel()

	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	_, err = manager.LoadWorldJSON("does_not_exist")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

// TestSerializationManager_GetSaveFileTimestamp はタイムスタンプ抽出のエラーパスを検証する。
func TestSerializationManager_GetSaveFileTimestamp(t *testing.T) {
	t.Parallel()

	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	t.Run("存在しないスロットはエラー", func(t *testing.T) {
		t.Parallel()
		_, err := manager.GetSaveFileTimestamp("does_not_exist")
		require.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("壊れたJSONはエラー", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, manager.storeSaveJSON("broken_json", []byte("not json")))
		_, err := manager.GetSaveFileTimestamp("broken_json")
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse save data")
	})
}

// TestSerializationManager_AutoSave_保存に失敗するとエラー は、セーブディレクトリが失われた状態で
// AutoSave を呼ぶと、内部の SaveWorld 失敗が "failed to auto save" として伝播することを検証する。
func TestSerializationManager_AutoSave_保存に失敗するとエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	dir := t.TempDir()
	manager, err := NewSerializationManager(WithSaveDir(dir))
	require.NoError(t, err)

	// この時点でディレクトリは空なので os.Remove で削除できる
	require.NoError(t, os.Remove(dir))

	err = manager.AutoSave(world)
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.ErrorContains(t, err, "failed to auto save")
}

// TestSerializationManager_GetSavePlayerName_プレイヤー名が空ならエラー は、
// playerName フィールドが空文字のセーブデータを弾くことを検証する。
func TestSerializationManager_GetSavePlayerName_プレイヤー名が空ならエラー(t *testing.T) {
	t.Parallel()

	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	require.NoError(t, manager.storeSaveJSON("no_player_name", []byte(`{"playerName":""}`)))

	_, err = manager.GetSavePlayerName("no_player_name")
	assert.EqualError(t, err, "player name not found in save data")
}

// TestSerializationManager_LoadWorld_不正なステージキーは復元を拒否する は、コンストラクタを
// 経由しない不正な StageKey が紛れ込んだセーブを validateStages が弾くことを検証する。
// ダンジョン階なのに深度0という、Validate が不正とみなす組み合わせを作る。
func TestSerializationManager_LoadWorld_不正なステージキーは復元を拒否する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	// Get が返すポインタは構造変更で無効化されるが、このテストでは後続の構造変更が無いため直接書き換えて問題ない
	dungeon := world.Components.Dungeon.Get(world.Resources.SingletonEntity)
	dungeon.CurrentStage = gc.StageKey{Name: "broken-dungeon", Depth: 0}

	require.NoError(t, manager.SaveWorld(world, "invalid_stage"))

	newWorld := testutil.InitTestWorld(t)
	err = manager.LoadWorld(newWorld, "invalid_stage")
	require.Error(t, err)
	assert.ErrorContains(t, err, "restored stage key is invalid")
}

// TestRestoreWorldFromJSON_PreservesWorldOnFailure はロードが復元段階で失敗しても、現行ワールドが
// 破壊されずに残ることを検証する。RestoreWorldFromJSON は本番ワールドを Reset する前に使い捨ての
// probe で全工程を検証するため、失敗は本番ワールドに波及しない。
func TestRestoreWorldFromJSON_PreservesWorldOnFailure(t *testing.T) {
	t.Parallel()

	manager, err := NewSerializationManager(WithSaveDir(t.TempDir()))
	require.NoError(t, err)

	// 復元段階で失敗するセーブを用意する。Dungeon を外すと reestablishSingleton が
	// シングルトンを特定できず失敗する。チェックサム・バージョンは通る
	bad := testutil.InitTestWorld(t)
	bad.Components.Dungeon.Remove(bad.Resources.SingletonEntity)
	require.NoError(t, manager.SaveWorld(bad, "bad_slot"))
	badJSON, err := manager.LoadWorldJSON("bad_slot")
	require.NoError(t, err)

	world := testutil.InitTestWorld(t)
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.Name.Add(player, &gc.Name{Name: "生存確認プレイヤー"})

	err = manager.RestoreWorldFromJSON(world, badJSON)
	require.Error(t, err)

	survived := false
	q := ecs.NewFilter1[gc.Player](world.ECS).Query()
	for q.Next() {
		if world.Components.Name.Get(q.Entity()).Name == "生存確認プレイヤー" {
			survived = true
		}
	}
	assert.True(t, survived, "ロード失敗時も現行ワールドは破壊されず保持される")
}
