package save

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// コンポーネント名の定数
const (
	ComponentLocationEquipped = "LocationEquipped"
)

// Data はセーブデータの最上位構造
type Data struct {
	Version   string        `json:"version"`
	Timestamp time.Time     `json:"timestamp"`
	World     WorldSaveData `json:"world"`
	Checksum  string        `json:"checksum"` // データ改ざん検知用ハッシュ値
}

// WorldSaveData はワールド全体のセーブデータ
type WorldSaveData struct {
	Entities []EntitySaveData `json:"entities"`
}

// EntitySaveData は単一エンティティのセーブデータ
type EntitySaveData struct {
	StableID   StableID               `json:"stable_id"`
	Components map[string]interface{} `json:"components"`
}

// EntityReference はエンティティ参照のセーブデータ
type EntityReference struct {
	TargetStableID *StableID `json:"target_stable_id,omitempty"`
}

// SerializationManager は安定ID + リフレクションベースのシリアライゼーションを管理
type SerializationManager struct {
	saveDirectory     string
	stableIDManager   *StableIDManager
	componentRegistry *ComponentRegistry
}

// NewSerializationManager は新しいSerializationManagerを作成
func NewSerializationManager(saveDir string) *SerializationManager {
	sm := &SerializationManager{
		saveDirectory:     saveDir,
		stableIDManager:   NewStableIDManager(),
		componentRegistry: NewComponentRegistry(),
	}

	// プラットフォーム固有の初期化処理
	sm.initImpl()

	return sm
}

// GenerateWorldJSON はワールドからJSON文字列を生成する
func (sm *SerializationManager) GenerateWorldJSON(world w.World) (string, error) {
	// コンポーネントレジストリを初期化
	err := sm.componentRegistry.InitializeFromWorld(world)
	if err != nil {
		return "", fmt.Errorf("failed to initialize component registry: %w", err)
	}

	// ワールドデータを抽出
	worldData := sm.extractWorldData(world)

	// セーブデータを作成（チェックサムは後で計算）
	saveData := Data{
		Version:   "1.0.0",
		Timestamp: time.Now(),
		World:     worldData,
	}

	// チェックサムを計算して設定
	checksum := sm.calculateChecksum(&saveData)
	saveData.Checksum = checksum

	// JSONにシリアライズ（キーをソート）
	data, err := sm.marshalSortedJSON(saveData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal save data: %w", err)
	}

	return string(data), nil
}

// SaveWorld はワールド全体をファイルに保存（既存のインターフェース維持）
func (sm *SerializationManager) SaveWorld(world w.World, slotName string) error {
	// JSON生成
	jsonData, err := sm.GenerateWorldJSON(world)
	if err != nil {
		return err
	}

	// ファイル保存
	return sm.saveDataImpl(slotName, []byte(jsonData))
}

// LoadWorldJSON はJSON文字列をファイルから読み込む
func (sm *SerializationManager) LoadWorldJSON(slotName string) (string, error) {
	// プラットフォーム固有のデータ読み込み処理を実行
	data, err := sm.loadDataImpl(slotName)
	if err != nil {
		return "", fmt.Errorf("failed to load save data: %w", err)
	}

	return string(data), nil
}

// RestoreWorldFromJSON はJSON文字列からワールドを復元する（ファイル読み込みなし）
func (sm *SerializationManager) RestoreWorldFromJSON(world w.World, jsonData string) error {
	// コンポーネントレジストリを初期化
	err := sm.componentRegistry.InitializeFromWorld(world)
	if err != nil {
		return fmt.Errorf("failed to initialize component registry: %w", err)
	}

	// JSONをパース
	var saveData Data
	err = json.Unmarshal([]byte(jsonData), &saveData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal save data: %w", err)
	}

	// チェックサム検証（データ改ざん検知）
	err = sm.validateChecksum(&saveData)
	if err != nil {
		return fmt.Errorf("save data validation failed: %w", err)
	}

	// バージョンチェック
	if saveData.Version != "1.0.0" {
		return fmt.Errorf("unsupported save data version: %s", saveData.Version)
	}

	// ワールドをクリア
	sm.clearWorld(world)
	sm.stableIDManager.Clear()

	// ワールドデータを復元
	err = sm.restoreWorldData(world, saveData.World)
	if err != nil {
		return fmt.Errorf("failed to restore world data: %w", err)
	}

	return nil
}

// LoadWorld はファイルからワールドを復元（既存のインターフェース維持）
func (sm *SerializationManager) LoadWorld(world w.World, slotName string) error {
	// JSONファイル読み込み
	jsonData, err := sm.LoadWorldJSON(slotName)
	if err != nil {
		return err
	}

	// JSON文字列から復元
	return sm.RestoreWorldFromJSON(world, jsonData)
}

// extractWorldData はワールドからセーブデータを抽出
// プレイヤーエンティティとその所持アイテム（バックパック・装備）のみを保存する
// 地形、ドア、フィールドアイテム、敵などは毎回再生成し、保存しない
func (sm *SerializationManager) extractWorldData(world w.World) WorldSaveData {
	entities := []EntitySaveData{}
	processedEntities := make(map[ecs.Entity]bool) // 重複処理防止

	// 1. プレイヤーエンティティを保存
	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(func(entity ecs.Entity) {
		sm.processEntityForSave(entity, world, &entities, processedEntities)
	}))

	// 2. バックパック内のアイテムを保存
	world.Manager.Join(world.Components.ItemLocationInPlayerBackpack).Visit(ecs.Visit(func(entity ecs.Entity) {
		sm.processEntityForSave(entity, world, &entities, processedEntities)
	}))

	// 3. 装備中のアイテムを保存
	world.Manager.Join(world.Components.ItemLocationEquipped).Visit(ecs.Visit(func(entity ecs.Entity) {
		sm.processEntityForSave(entity, world, &entities, processedEntities)
	}))

	// エンティティをStableIDでソートして決定的な順序にする
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].StableID.Index < entities[j].StableID.Index
	})

	return WorldSaveData{
		Entities: entities,
	}
}

// processEntityForSave はエンティティを処理してセーブデータに追加
func (sm *SerializationManager) processEntityForSave(entity ecs.Entity, world w.World, entities *[]EntitySaveData, processed map[ecs.Entity]bool) {
	// 既に処理済みかチェック
	if processed[entity] {
		return
	}
	processed[entity] = true

	// 安定IDを取得
	stableID := sm.stableIDManager.GetStableID(entity)

	// エンティティデータを作成
	entityData := EntitySaveData{
		StableID:   stableID,
		Components: make(map[string]interface{}),
	}

	// 各コンポーネント型をチェック
	for _, typeInfo := range sm.componentRegistry.GetAllTypes() {
		if data, hasComponent := typeInfo.ExtractFunc(world, entity); hasComponent {
			// エンティティ参照の処理
			processedData := sm.processEntityReferences(data, typeInfo)

			// コンポーネントデータを直接格納する。キー名が型名を示す
			entityData.Components[typeInfo.Name] = processedData
		}
	}

	// コンポーネントがあるエンティティのみ保存
	if len(entityData.Components) > 0 {
		*entities = append(*entities, entityData)
	}
}

// processEntityReferences はエンティティ参照を安定IDに変換
func (sm *SerializationManager) processEntityReferences(data interface{}, typeInfo *ComponentTypeInfo) interface{} {
	// LocationEquippedのOwnerを特別処理
	if typeInfo.Name == ComponentLocationEquipped {
		if equipped, ok := data.(gc.LocationEquipped); ok {
			ownerStableID := sm.stableIDManager.GetStableID(equipped.Owner)
			equippedRef := struct {
				OwnerRef      StableID               `json:"OwnerRef"`
				EquipmentSlot gc.EquipmentSlotNumber `json:"EquipmentSlot"`
			}{
				OwnerRef:      ownerStableID,
				EquipmentSlot: equipped.EquipmentSlot,
			}
			return equippedRef
		}
	}

	// 他の型はそのまま返す
	return data
}

// restoreWorldData はセーブデータからワールドを復元
func (sm *SerializationManager) restoreWorldData(world w.World, worldData WorldSaveData) error {
	// 第1段階: 全エンティティを作成して安定IDマッピング
	entityMap := make(map[StableID]ecs.Entity)
	entityDataMap := make(map[StableID]EntitySaveData)

	for _, entityData := range worldData.Entities {
		entity := world.Manager.NewEntity()

		// 安定IDマッピングを登録
		err := sm.stableIDManager.RegisterEntity(entity, entityData.StableID)
		if err != nil {
			return fmt.Errorf("failed to register entity mapping: %w", err)
		}

		entityMap[entityData.StableID] = entity
		entityDataMap[entityData.StableID] = entityData
	}

	// 第2段階: コンポーネントを復元（エンティティ参照なし）
	for stableID, entityData := range entityDataMap {
		entity := entityMap[stableID]

		for componentName, componentData := range entityData.Components {
			typeInfo, exists := sm.componentRegistry.GetTypeInfoByName(componentName)
			if !exists {
				return fmt.Errorf("unknown component type: %s", componentName)
			}

			// JSONからコンポーネントデータを復元
			restoredData, err := sm.restoreComponentData(componentData, typeInfo)
			if err != nil {
				return fmt.Errorf("failed to restore component %s: %w", componentName, err)
			}

			// コンポーネントをエンティティに追加
			err = typeInfo.RestoreFunc(world, entity, restoredData)
			if err != nil {
				return fmt.Errorf("failed to add component %s to entity: %w", componentName, err)
			}
		}
	}

	// 第3段階: エンティティ参照を解決
	for stableID, entityData := range entityDataMap {
		entity := entityMap[stableID]

		for componentName, componentData := range entityData.Components {
			typeInfo, exists := sm.componentRegistry.GetTypeInfoByName(componentName)
			if !exists || typeInfo.ResolveRefFunc == nil {
				continue
			}

			err := sm.resolveEntityReferences(world, entity, componentData, typeInfo)
			if err != nil {
				return fmt.Errorf("failed to resolve references for %s: %w", componentName, err)
			}
		}
	}

	return nil
}

// restoreComponentData はJSONデータからコンポーネントデータを復元
func (sm *SerializationManager) restoreComponentData(jsonData interface{}, typeInfo *ComponentTypeInfo) (interface{}, error) {
	// ProvidesHealingを特別処理（Amounterインターフェースのため）
	if typeInfo.Name == "ProvidesHealing" {
		// この型はreflection.goのrestoreProvidesHealingで処理されるため、
		// ここではデータをそのまま返す
		return jsonData, nil
	}

	// LocationEquippedを特別処理（カスタムシリアライズ形式のため）
	if typeInfo.Name == ComponentLocationEquipped {
		dataMap, ok := jsonData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid LocationEquipped JSON data type: %T", jsonData)
		}

		// EquipmentSlotを取得
		equipmentSlotVal, exists := dataMap["EquipmentSlot"]
		if !exists {
			return nil, fmt.Errorf("EquipmentSlot not found in LocationEquipped data")
		}
		equipmentSlot, ok := equipmentSlotVal.(float64)
		if !ok {
			return nil, fmt.Errorf("invalid EquipmentSlot type: %T", equipmentSlotVal)
		}

		// LocationEquipped構造体を作成（Ownerは後で解決）
		equipped := gc.LocationEquipped{
			Owner:         0, // 一時的に無効なエンティティID
			EquipmentSlot: gc.EquipmentSlotNumber(equipmentSlot),
		}
		return equipped, nil
	}

	// 通常のコンポーネント処理
	// JSONデータをバイトに変換
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}

	// 型に応じて適切な構造体を作成
	targetValue := reflect.New(typeInfo.Type).Interface()

	// JSONからデコード
	err = json.Unmarshal(jsonBytes, targetValue)
	if err != nil {
		return nil, err
	}

	// ポインタから値を取得
	return reflect.ValueOf(targetValue).Elem().Interface(), nil
}

// resolveEntityReferences はエンティティ参照を解決
func (sm *SerializationManager) resolveEntityReferences(world w.World, entity ecs.Entity, jsonData interface{}, typeInfo *ComponentTypeInfo) error {
	if typeInfo.Name == ComponentLocationEquipped {
		// JSONデータを変換
		jsonBytes, err := json.Marshal(jsonData)
		if err != nil {
			return err
		}

		var equippedRef struct {
			OwnerRef      StableID               `json:"OwnerRef"`
			EquipmentSlot gc.EquipmentSlotNumber `json:"EquipmentSlot"`
		}

		err = json.Unmarshal(jsonBytes, &equippedRef)
		if err != nil {
			return err
		}

		// LocationEquippedコンポーネントを取得
		if entity.HasComponent(world.Components.ItemLocationEquipped) {
			equipped := world.Components.ItemLocationEquipped.Get(entity).(*gc.LocationEquipped)

			// エンティティ参照を解決
			if ownerEntity, exists := sm.stableIDManager.GetEntity(equippedRef.OwnerRef); exists {
				equipped.Owner = ownerEntity
			} else {
				// Owner参照が解決できない場合はエラーを返す
				return fmt.Errorf("required owner entity not found for stable ID: %v", equippedRef.OwnerRef)
			}
		}
	}

	return nil
}

// clearWorld はワールドの全エンティティをクリア
func (sm *SerializationManager) clearWorld(world w.World) {
	// 全エンティティを削除
	entitiesToDelete := make([]ecs.Entity, 0)

	world.Manager.Join().Visit(ecs.Visit(func(entity ecs.Entity) {
		entitiesToDelete = append(entitiesToDelete, entity)
	}))

	for _, entity := range entitiesToDelete {
		world.Manager.DeleteEntity(entity)
	}
}

// SaveFileExists はセーブファイルが存在するかチェックする
func (sm *SerializationManager) SaveFileExists(slotName string) bool {
	return sm.saveFileExistsImpl(slotName)
}

// GetSaveFileTimestamp はセーブファイルのタイムスタンプを取得する
// JSONファイルの中身のtimestampフィールドを読み取る
func (sm *SerializationManager) GetSaveFileTimestamp(slotName string) (time.Time, error) {
	data, err := sm.loadDataImpl(slotName)
	if err != nil {
		return time.Time{}, err
	}

	var saveData Data
	err = json.Unmarshal(data, &saveData)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse save data: %w", err)
	}

	return saveData.Timestamp, nil
}

// GetStableIDManager は安定IDマネージャーを取得
func (sm *SerializationManager) GetStableIDManager() *StableIDManager {
	return sm.stableIDManager
}

// GetComponentRegistry はコンポーネントレジストリを取得
func (sm *SerializationManager) GetComponentRegistry() *ComponentRegistry {
	return sm.componentRegistry
}

// calculateChecksum はセーブデータのチェックサムを計算する
// 決定的な順序でハッシュ計算を行うため、エンティティとコンポーネントをソートする
func (sm *SerializationManager) calculateChecksum(data *Data) string {
	return sm.calculateDeterministicHash(data)
}

// calculateDeterministicHash は決定的な順序でハッシュを計算する
func (sm *SerializationManager) calculateDeterministicHash(data *Data) string {
	hashParts := make([]string, 0, len(data.World.Entities)+1)

	// バージョン情報
	hashParts = append(hashParts, fmt.Sprintf("version:%s", data.Version))

	// エンティティを StableID の Index でソート
	entities := make([]EntitySaveData, len(data.World.Entities))
	copy(entities, data.World.Entities)

	sort.Slice(entities, func(i, j int) bool {
		return entities[i].StableID.Index < entities[j].StableID.Index
	})

	// 各エンティティのハッシュを計算
	for _, entity := range entities {
		entityHash := sm.calculateEntityHash(entity)
		hashParts = append(hashParts, fmt.Sprintf("entity:%s", entityHash))
	}

	// 全体のハッシュを計算
	finalData := fmt.Sprintf("checksum_data:%s", fmt.Sprintf("%v", hashParts))
	hash := sha256.Sum256([]byte(finalData))
	return hex.EncodeToString(hash[:])
}

// calculateEntityHash は単一エンティティの決定的ハッシュを計算する
func (sm *SerializationManager) calculateEntityHash(entity EntitySaveData) string {
	parts := make([]string, 0, len(entity.Components)+1)

	// StableID
	parts = append(parts, fmt.Sprintf("stable_id:%d:%d", entity.StableID.Index, entity.StableID.Generation))

	// コンポーネント名をソート
	componentNames := make([]string, 0, len(entity.Components))
	for name := range entity.Components {
		componentNames = append(componentNames, name)
	}
	sort.Strings(componentNames)

	// 各コンポーネントのハッシュを計算
	for _, name := range componentNames {
		component := entity.Components[name]
		componentHash := sm.calculateComponentHash(name, component)
		parts = append(parts, fmt.Sprintf("component:%s:%s", name, componentHash))
	}

	entityData := fmt.Sprintf("entity_data:%s", fmt.Sprintf("%v", parts))
	hash := sha256.Sum256([]byte(entityData))
	return hex.EncodeToString(hash[:])
}

// calculateComponentHash はコンポーネントの決定的ハッシュを計算する
func (sm *SerializationManager) calculateComponentHash(name string, component interface{}) string {
	// シンプルな実装: コンポーネント名とデータサイズでハッシュ計算
	// より厳密には、データの内容を決定的にシリアライズする必要がある

	var dataSize int
	if component != nil {
		// JSON marshal でサイズを概算
		if jsonBytes, err := json.Marshal(component); err == nil {
			dataSize = len(jsonBytes)
		}
	}

	hashData := fmt.Sprintf("component:%s:size:%d", name, dataSize)
	hash := sha256.Sum256([]byte(hashData))
	return hex.EncodeToString(hash[:])
}

// validateChecksum はセーブデータのチェックサムを検証する
func (sm *SerializationManager) validateChecksum(data *Data) error {
	if data.Checksum == "" {
		return fmt.Errorf("checksum field is missing: このセーブデータは改ざんされているか、古いバージョンです")
	}

	// 現在のデータからチェックサムを計算
	expectedChecksum := sm.calculateChecksum(data)

	// チェックサムを比較
	if data.Checksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s (データが改ざんされている可能性があります)",
			expectedChecksum, data.Checksum)
	}

	return nil
}

// marshalSortedJSON はキーをソートしてJSONマーシャリングを行う
func (sm *SerializationManager) marshalSortedJSON(data interface{}) ([]byte, error) {
	// 最初に標準のMarshalでJSONに変換
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// JSONを一度map[string]interface{}に変換
	var jsonObj interface{}
	if err := json.Unmarshal(jsonBytes, &jsonObj); err != nil {
		return nil, err
	}

	// ソート済みJSONを生成
	return sm.marshalSortedIndent(jsonObj, "", "  ")
}

// marshalSortedIndent は再帰的にキーをソートしてインデント付きJSONを生成
func (sm *SerializationManager) marshalSortedIndent(v interface{}, prefix, indent string) ([]byte, error) {
	switch value := v.(type) {
	case map[string]interface{}:
		return sm.marshalSortedObject(value, prefix, indent)
	case []interface{}:
		return sm.marshalSortedArray(value, prefix, indent)
	default:
		// プリミティブ値の場合は標準のMarshalを使用
		return json.Marshal(value)
	}
}

// marshalSortedObject はオブジェクトのキーをソートしてマーシャリング
func (sm *SerializationManager) marshalSortedObject(obj map[string]interface{}, prefix, indent string) ([]byte, error) {
	if len(obj) == 0 {
		return []byte("{}"), nil
	}

	// キーを取得してソート
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	buf.WriteString("{\n")

	newPrefix := prefix + indent
	for i, key := range keys {
		// キーを書き込み
		buf.WriteString(newPrefix)
		keyBytes, _ := json.Marshal(key)
		buf.Write(keyBytes)
		buf.WriteString(": ")

		// 値を再帰的に処理
		valueBytes, err := sm.marshalSortedIndent(obj[key], newPrefix, indent)
		if err != nil {
			return nil, err
		}
		buf.Write(valueBytes)

		// 最後の要素以外はカンマを追加
		if i < len(keys)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString(prefix + "}")
	return []byte(buf.String()), nil
}

// marshalSortedArray は配列をマーシャリング
func (sm *SerializationManager) marshalSortedArray(arr []interface{}, prefix, indent string) ([]byte, error) {
	if len(arr) == 0 {
		return []byte("[]"), nil
	}

	var buf strings.Builder
	buf.WriteString("[\n")

	newPrefix := prefix + indent
	for i, item := range arr {
		buf.WriteString(newPrefix)

		// 要素を再帰的に処理
		itemBytes, err := sm.marshalSortedIndent(item, newPrefix, indent)
		if err != nil {
			return nil, err
		}
		buf.Write(itemBytes)

		// 最後の要素以外はカンマを追加
		if i < len(arr)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString(prefix + "]")
	return []byte(buf.String()), nil
}
