package save

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

const saveDataVersion = "2.0.0"

const maxAutoSaves = 4

// autoSavePrefix はオートセーブスロット名の接頭辞
const autoSavePrefix = "auto_"

const defaultSaveDir = "./saves"

// Option はSerializationManagerの設定を変更する関数
type Option func(*SerializationManager)

// WithSaveDir はセーブディレクトリを変更する
func WithSaveDir(dir string) Option {
	return func(sm *SerializationManager) {
		sm.saveDirectory = dir
	}
}

// SerializationManager は ark-serde ベースのワールドシリアライゼーションを管理する
type SerializationManager struct {
	saveDirectory string
}

// NewSerializationManager は新しいSerializationManagerを作成する
func NewSerializationManager(opts ...Option) (*SerializationManager, error) {
	sm := &SerializationManager{
		saveDirectory: defaultSaveDir,
	}
	for _, opt := range opts {
		opt(sm)
	}
	if err := sm.initImpl(); err != nil {
		return nil, err
	}
	return sm, nil
}

// GenerateWorldJSON はワールドからJSON文字列を生成する
func (sm *SerializationManager) GenerateWorldJSON(world w.World) (string, error) {
	worldJSON, err := serializeWorld(world)
	if err != nil {
		return "", fmt.Errorf("failed to serialize world: %w", err)
	}

	env := saveEnvelope{
		Version:    saveDataVersion,
		Timestamp:  time.Now(),
		PlayerName: extractPlayerName(world),
		World:      worldJSON,
	}
	env.Checksum = checksumOf(&env)

	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal save data: %w", err)
	}
	return string(data), nil
}

// SaveWorld はワールド全体をファイルに保存する
func (sm *SerializationManager) SaveWorld(world w.World, slotName string) error {
	jsonData, err := sm.GenerateWorldJSON(world)
	if err != nil {
		return err
	}
	return sm.saveDataImpl(slotName, []byte(jsonData))
}

// LoadWorldJSON はJSON文字列をファイルから読み込む
func (sm *SerializationManager) LoadWorldJSON(slotName string) (string, error) {
	data, err := sm.loadDataImpl(slotName)
	if err != nil {
		return "", fmt.Errorf("failed to load save data: %w", err)
	}
	return string(data), nil
}

// RestoreWorldFromJSON はJSON文字列からワールドを復元する
func (sm *SerializationManager) RestoreWorldFromJSON(world w.World, jsonData string) error {
	var env saveEnvelope
	if err := json.Unmarshal([]byte(jsonData), &env); err != nil {
		return fmt.Errorf("failed to unmarshal save data: %w", err)
	}

	if err := validateChecksum(&env); err != nil {
		return fmt.Errorf("save data validation failed: %w", err)
	}

	if env.Version != saveDataVersion {
		return fmt.Errorf("unsupported save data version: %s", env.Version)
	}

	// 本番ワールドを破壊する前に、使い捨ての probe ワールドで復元の全工程を検証する。
	// ark-serde の Deserialize はリセット済みワールドを要求し、途中で失敗しても巻き戻せない。
	// 先に probe を通し、成功したときだけ本番ワールドへ適用することで、ロード失敗時も
	// 本番ワールドを無傷で残し、呼び出し側がアプリを継続できるようにする。
	// 復元を2回走らせるコストは、ロードがゲーム開始時の1回きりの操作なので許容する。
	probe, err := w.InitWorld(&gc.Components{}, world.Config)
	if err != nil {
		return fmt.Errorf("failed to create probe world: %w", err)
	}
	if err := restoreInto(probe, env.World); err != nil {
		return err
	}

	return restoreInto(world, env.World)
}

// restoreInto はリセット済みワールドへ復元の全工程を適用する。deserialize は事前の
// world.ECS.Reset() を要求する。probe と本番ワールドの両方でこの手順を共有する。
func restoreInto(world w.World, worldJSON []byte) error {
	// ark-serdeのDeserializeはリセット済みワールドを要求する
	world.ECS.Reset()

	if err := deserializeWorld(world, worldJSON); err != nil {
		return fmt.Errorf("failed to restore world data: %w", err)
	}

	// スキップした一時コンポーネントとシングルトン参照を再確立する
	if err := reestablishSingleton(world); err != nil {
		return fmt.Errorf("failed to reestablish singleton: %w", err)
	}

	// 信頼境界。復元したステージキーの整合を検査する
	if err := validateStages(world); err != nil {
		return fmt.Errorf("failed to validate stages: %w", err)
	}
	return nil
}

// LoadWorld はファイルからワールドを復元する
func (sm *SerializationManager) LoadWorld(world w.World, slotName string) error {
	jsonData, err := sm.LoadWorldJSON(slotName)
	if err != nil {
		return err
	}
	return sm.RestoreWorldFromJSON(world, jsonData)
}

// SaveFileExists はセーブファイルが存在するかチェックする
func (sm *SerializationManager) SaveFileExists(slotName string) bool {
	return sm.saveFileExistsImpl(slotName)
}

// GetSaveFileTimestamp はセーブファイルのタイムスタンプを取得する。
// セーブデータ全体をデシリアライズせず、タイムスタンプだけを抽出する。
func (sm *SerializationManager) GetSaveFileTimestamp(slotName string) (time.Time, error) {
	data, err := sm.loadDataImpl(slotName)
	if err != nil {
		return time.Time{}, err
	}
	var partial struct {
		Timestamp time.Time `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &partial); err != nil {
		return time.Time{}, fmt.Errorf("failed to parse save data: %w", err)
	}
	return partial.Timestamp, nil
}

// ListSaves はセーブデータの一覧を新しい順に返す
func (sm *SerializationManager) ListSaves() ([]string, error) {
	names, err := sm.listSavesImpl()
	if err != nil {
		return nil, err
	}

	// タイムスタンプを取得できたもののみ返す
	var valid []string
	timestamps := make(map[string]time.Time, len(names))
	for _, name := range names {
		ts, err := sm.GetSaveFileTimestamp(name)
		if err != nil {
			continue
		}
		valid = append(valid, name)
		timestamps[name] = ts
	}

	slices.SortFunc(valid, func(a, b string) int {
		return timestamps[b].Compare(timestamps[a])
	})
	return valid, nil
}

// ListAutoSaves はオートセーブスロット名の一覧を返す。
func (sm *SerializationManager) ListAutoSaves() ([]string, error) {
	saves, err := sm.ListSaves()
	if err != nil {
		return nil, err
	}
	var autoSaves []string
	for _, name := range saves {
		if strings.HasPrefix(name, autoSavePrefix) {
			autoSaves = append(autoSaves, name)
		}
	}
	return autoSaves, nil
}

// AutoSave はオートセーブを実行する。
// スロット名の生成、保存、古いオートセーブのローテーションを一括で行う。
func (sm *SerializationManager) AutoSave(world w.World) error {
	slotName := fmt.Sprintf("%s%d", autoSavePrefix, time.Now().UnixNano())
	if err := sm.SaveWorld(world, slotName); err != nil {
		return fmt.Errorf("failed to auto save: %w", err)
	}
	if err := sm.rotateAutoSaves(); err != nil {
		return fmt.Errorf("failed to delete old auto save: %w", err)
	}
	return nil
}

// rotateAutoSaves はオートセーブを最大件数まで削減する。
// 古い順に削除して maxAutoSaves 件を保持する。
func (sm *SerializationManager) rotateAutoSaves() error {
	autoSaves, err := sm.ListAutoSaves()
	if err != nil {
		return err
	}

	if len(autoSaves) <= maxAutoSaves {
		return nil
	}

	for _, name := range autoSaves[maxAutoSaves:] {
		if err := sm.deleteSaveImpl(name); err != nil {
			return fmt.Errorf("failed to prune auto save %s: %w", name, err)
		}
	}
	return nil
}

// GetSavePlayerName はセーブデータからプレイヤー名を取得する。
// セーブデータ全体をデシリアライズせず、封筒のメタ情報だけを読む。
func (sm *SerializationManager) GetSavePlayerName(slotName string) (string, error) {
	data, err := sm.loadDataImpl(slotName)
	if err != nil {
		return "", err
	}
	var partial struct {
		PlayerName string `json:"playerName"`
	}
	if err := json.Unmarshal(data, &partial); err != nil {
		return "", fmt.Errorf("failed to parse save data: %w", err)
	}
	if partial.PlayerName == "" {
		return "", fmt.Errorf("player name not found in save data")
	}
	return partial.PlayerName, nil
}

// validateChecksum はセーブデータのチェックサムを検証する
func validateChecksum(env *saveEnvelope) error {
	if env.Checksum == "" {
		return fmt.Errorf("checksum field is missing: this save data is tampered or from an old version")
	}
	expected := checksumOf(env)
	if env.Checksum != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s; data may be tampered",
			expected, env.Checksum)
	}
	return nil
}
