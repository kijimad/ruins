package save

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

const saveDataVersion = "1.0.0"

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

// SerializationManager は安定ID + 静的型ベースのシリアライゼーションを管理する
type SerializationManager struct {
	saveDirectory   string
	stableIDManager *StableIDManager
}

// NewSerializationManager は新しいSerializationManagerを作成する
func NewSerializationManager(opts ...Option) (*SerializationManager, error) {
	sm := &SerializationManager{
		saveDirectory:   defaultSaveDir,
		stableIDManager: NewStableIDManager(),
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
	worldData := sm.extractWorldData(world)

	saveData := oapi.SaveDataSaveData{
		Version:   saveDataVersion,
		Timestamp: time.Now(),
		World:     worldData,
	}

	checksum, err := sm.calculateChecksum(&saveData)
	if err != nil {
		return "", err
	}
	saveData.Checksum = checksum

	data, err := json.MarshalIndent(saveData, "", "  ")
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
	// OpenAPIスキーマでバリデーション
	if err := ValidateSaveJSON(jsonData); err != nil {
		return err
	}

	var saveData oapi.SaveDataSaveData
	if err := json.Unmarshal([]byte(jsonData), &saveData); err != nil {
		return fmt.Errorf("failed to unmarshal save data: %w", err)
	}

	if err := sm.validateChecksum(&saveData); err != nil {
		return fmt.Errorf("save data validation failed: %w", err)
	}

	if string(saveData.Version) != saveDataVersion {
		return fmt.Errorf("unsupported save data version: %s", saveData.Version)
	}

	world.Manager.DeleteAllEntities()
	world.InitSingleton()
	sm.stableIDManager.Clear()

	if err := sm.restoreWorldData(world, saveData.World); err != nil {
		return fmt.Errorf("failed to restore world data: %w", err)
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

// extractWorldData はワールドからセーブデータを抽出する。
// プレイヤーエンティティとその所持アイテム（バックパック・装備）のみを保存する。
// 地形、扉、フィールドアイテム、敵などは毎回再生成し、保存しない。
func (sm *SerializationManager) extractWorldData(world w.World) oapi.SaveDataWorldSaveData {
	entities := []oapi.SaveDataEntitySaveData{}
	processed := make(map[ecs.Entity]bool)

	collect := func(entity ecs.Entity) {
		if processed[entity] {
			return
		}
		processed[entity] = true
		entityData := sm.extractEntity(entity, world)
		entities = append(entities, entityData)
	}

	// 1. プレイヤーエンティティ
	playerQuery := ecs.NewFilter1[gc.Player](world.World).Query()
	for playerQuery.Next() {
		collect(playerQuery.Entity())
	}
	// 2. バックパック内アイテム
	backpackQuery := ecs.NewFilter1[gc.LocationInBackpack](world.World).Query()
	for backpackQuery.Next() {
		collect(backpackQuery.Entity())
	}
	// 3. 装備中アイテム
	equippedQuery := ecs.NewFilter1[gc.LocationEquipped](world.World).Query()
	for equippedQuery.Next() {
		collect(equippedQuery.Entity())
	}
	// 4. 隊員エンティティ（死亡していないもの）
	squadQuery := ecs.NewFilter1[gc.SquadMember](world.World).Query()
	for squadQuery.Next() {
		entity := squadQuery.Entity()
		if !world.Components.Dead.Has(entity) {
			collect(entity)
		}
	}

	sort.Slice(entities, func(i, j int) bool {
		return entities[i].StableId.Index < entities[j].StableId.Index
	})

	return oapi.SaveDataWorldSaveData{
		Entities:     entities,
		GameProgress: gameProgressToSaveData(query.GetGameProgress(world)),
	}
}

// extractMarkers はマーカーコンポーネントとStackableを抽出する
func (sm *SerializationManager) extractMarkers(entity ecs.Entity, c *gc.Components, comp *oapi.SaveDataComponentsMap) {
	if c.Player.Has(entity) {
		comp.Player = emptyMarker()
	}
	if c.FactionAlly.Has(entity) {
		comp.FactionAllyData = emptyMarker()
	}
	if c.LocationInBackpack.Has(entity) {
		backpack := c.LocationInBackpack.Get(entity)
		ownerStableID := sm.stableIDManager.GetStableID(backpack.Owner)
		comp.LocationInBackpack = &oapi.SaveDataLocationInBackpackComponent{
			OwnerRef: stableIDToSaveData(ownerStableID),
		}
	}
	if c.StatsChanged.Has(entity) {
		comp.StatsChanged = emptyMarker()
	}
	if c.Stackable.Has(entity) {
		stackable := c.Stackable.Get(entity)
		m := oapi.SaveDataMarkerComponent{"Count": stackable.Count}
		comp.Stackable = &m
	}
}

// extractEntity はエンティティのコンポーネントをSaveData型に変換する
func (sm *SerializationManager) extractEntity(entity ecs.Entity, world w.World) oapi.SaveDataEntitySaveData {
	stableID := sm.stableIDManager.GetStableID(entity)
	c := world.Components
	comp := oapi.SaveDataComponentsMap{}

	sm.extractMarkers(entity, c, &comp)

	// データコンポーネント
	if c.Name.Has(entity) {
		name := c.Name.Get(entity)
		comp.Name = &oapi.SaveDataNameComponent{Name: name.Name}
	}
	if c.Description.Has(entity) {
		desc := c.Description.Get(entity)
		comp.Description = &oapi.SaveDataDescriptionComponent{Description: desc.Description}
	}
	if c.HP.Has(entity) {
		sd := hpToSaveData(*c.HP.Get(entity))
		comp.HP = &sd
	}
	if c.WeightCapacity.Has(entity) {
		sd := weightCapacityToSaveData(*c.WeightCapacity.Get(entity))
		comp.WeightCapacity = &sd
	}
	if c.TurnBased.Has(entity) {
		tb := c.TurnBased.Get(entity)
		sd := turnBasedToSaveData(*tb)
		comp.TurnBased = &sd
	}
	if c.Abilities.Has(entity) {
		ab := c.Abilities.Get(entity)
		sd := abilitiesToSaveData(*ab)
		comp.Abilities = &sd
	}
	if c.HealthStatus.Has(entity) {
		hs := c.HealthStatus.Get(entity)
		sd := healthStatusToSaveData(*hs)
		comp.HealthStatus = &sd
	}
	if c.Skills.Has(entity) {
		sk := c.Skills.Get(entity)
		sd := skillsToSaveData(*sk)
		comp.Skills = &sd
	}

	// 表示コンポーネント
	if c.Camera.Has(entity) {
		cam := c.Camera.Get(entity)
		sd := cameraToSaveData(*cam)
		comp.Camera = &sd
	}
	if c.GridElement.Has(entity) {
		ge := c.GridElement.Get(entity)
		sd := gridElementToSaveData(*ge)
		comp.GridElement = &sd
	}
	if c.SpriteRender.Has(entity) {
		sr := c.SpriteRender.Get(entity)
		sd := spriteRenderToSaveData(*sr)
		comp.SpriteRender = &sd
	}
	if c.LightSource.Has(entity) {
		ls := c.LightSource.Get(entity)
		sd := lightSourceToSaveData(*ls)
		comp.LightSource = &sd
	}

	// アイテム属性コンポーネント
	if c.Wearable.Has(entity) {
		w := c.Wearable.Get(entity)
		sd := wearableToSaveData(*w)
		comp.Wearable = &sd
	}
	if c.Value.Has(entity) {
		v := c.Value.Get(entity)
		comp.Value = &oapi.SaveDataValueComponent{Value: int32(v.Value)}
	}
	if c.Melee.Has(entity) {
		m := c.Melee.Get(entity)
		sd := meleeToSaveData(*m)
		comp.Melee = &sd
	}
	if c.Fire.Has(entity) {
		f := c.Fire.Get(entity)
		sd := fireToSaveData(*f)
		comp.Fire = &sd
	}
	if c.Recipe.Has(entity) {
		r := c.Recipe.Get(entity)
		sd := recipeToSaveData(*r)
		comp.Recipe = &sd
	}
	if c.Ammo.Has(entity) {
		a := c.Ammo.Get(entity)
		sd := ammoToSaveData(*a)
		comp.Ammo = &sd
	}

	// アイテム効果コンポーネント
	if c.Consumable.Has(entity) {
		con := c.Consumable.Get(entity)
		sd := consumableToSaveData(*con)
		comp.Consumable = &sd
	}
	if c.ProvidesHealing.Has(entity) {
		ph := c.ProvidesHealing.Get(entity)
		sd := providesHealingToSaveData(*ph)
		comp.ProvidesHealing = &sd
	}
	if c.ProvidesNutrition.Has(entity) {
		pn := c.ProvidesNutrition.Get(entity)
		comp.ProvidesNutrition = &oapi.SaveDataProvidesNutritionComponent{Amount: int32(pn.Amount)}
	}
	if c.InflictsDamage.Has(entity) {
		id := c.InflictsDamage.Get(entity)
		comp.InflictsDamage = &oapi.SaveDataInflictsDamageComponent{Amount: int32(id.Amount)}
	}
	if c.Wallet.Has(entity) {
		wal := c.Wallet.Get(entity)
		comp.Wallet = &oapi.SaveDataWalletComponent{Currency: int32(wal.Currency)}
	}

	// 戦闘コンポーネント
	if c.CommandTable.Has(entity) {
		ct := c.CommandTable.Get(entity)
		comp.CommandTable = &oapi.SaveDataCommandTableComponent{Name: ct.Name}
	}

	// 隊員コンポーネント
	if c.SquadMember.Has(entity) {
		comp.SquadMember = &oapi.SaveDataSquadMemberComponent{}
	}
	if c.SoloAI.Has(entity) {
		ai := c.SoloAI.Get(entity)
		sd := soloAIToSaveData(*ai)
		comp.SquadPolicy = &sd
	}
	if c.SquadAI.Has(entity) {
		ai := c.SquadAI.Get(entity)
		sd := squadAIToSaveData(*ai)
		comp.SquadPolicy = &sd
	}
	// エンティティ参照コンポーネント (LocationEquipped)
	if c.LocationEquipped.Has(entity) {
		equipped := c.LocationEquipped.Get(entity)
		ownerStableID := sm.stableIDManager.GetStableID(equipped.Owner)
		comp.LocationEquipped = &oapi.SaveDataLocationEquippedComponent{
			OwnerRef:      stableIDToSaveData(ownerStableID),
			EquipmentSlot: int32(equipped.EquipmentSlot),
		}
	}

	return oapi.SaveDataEntitySaveData{
		StableId:   stableIDToSaveData(stableID),
		Components: comp,
	}
}

// entityEntry はエンティティ復元時の一時データ。
// dataは大きな構造体のため、コピーを避けてworldData.Entitiesの要素を指すポインタで保持する
type entityEntry struct {
	entity ecs.Entity
	data   *oapi.SaveDataEntitySaveData
}

// restoreWorldData はセーブデータからワールドを復元する
func (sm *SerializationManager) restoreWorldData(world w.World, worldData oapi.SaveDataWorldSaveData) error {
	// 第1段階: 全エンティティを作成して安定IDマッピング
	entries := make([]entityEntry, 0, len(worldData.Entities))
	for i := range worldData.Entities {
		entityData := &worldData.Entities[i]
		entity := world.World.NewEntity()
		stableID := stableIDFromSaveData(entityData.StableId)
		if err := sm.stableIDManager.RegisterEntity(entity, stableID); err != nil {
			return fmt.Errorf("failed to register entity mapping: %w", err)
		}
		entries = append(entries, entityEntry{entity: entity, data: entityData})
	}

	// 第2段階: コンポーネントを復元
	c := world.Components
	for _, entry := range entries {
		restoreComponents(entry.entity, entry.data.Components, c)
	}

	// 第3段階: エンティティ参照を解決
	for _, entry := range entries {
		comp := entry.data.Components

		// LocationInBackpack の Owner を解決
		if comp.LocationInBackpack != nil {
			ownerStableID := stableIDFromSaveData(comp.LocationInBackpack.OwnerRef)
			ownerEntity, exists := sm.stableIDManager.GetEntity(ownerStableID)
			if !exists {
				return fmt.Errorf("required owner entity not found for stable ID: %v", ownerStableID)
			}
			backpack := c.LocationInBackpack.Get(entry.entity)
			backpack.Owner = ownerEntity
		}

		// LocationEquipped の Owner を解決
		if comp.LocationEquipped != nil {
			ownerStableID := stableIDFromSaveData(comp.LocationEquipped.OwnerRef)
			ownerEntity, exists := sm.stableIDManager.GetEntity(ownerStableID)
			if !exists {
				return fmt.Errorf("required owner entity not found for stable ID: %v", ownerStableID)
			}
			equipped := c.LocationEquipped.Get(entry.entity)
			equipped.Owner = ownerEntity
		}
	}

	// 第4段階: 派生コンポーネントの再計算をマークする
	for _, entry := range entries {
		if entry.c.Skills.Has(entity) {
			entry.c.StatsChanged.Add(entity, &gc.StatsChanged{})
		}
	}

	// GameProgressを復元
	if gp := gameProgressFromSaveData(worldData.GameProgress); gp != nil {
		*query.GetGameProgress(world) = *gp
	}

	return nil
}

// restoreComponents はComponentsMapから全コンポーネントをエンティティに復元する
func restoreComponents(entity ecs.Entity, comp oapi.SaveDataComponentsMap, c *gc.Components) {
	// マーカーコンポーネント (NullComponent)
	if comp.Player != nil {
		c.Player.Add(entity, &gc.Player{})
	}
	if comp.FactionAllyData != nil {
		c.FactionAlly.Add(entity, &gc.FactionAllyData{})
	}
	// LocationInBackpack（Ownerは第3段階で解決）
	if comp.LocationInBackpack != nil {
		c.LocationInBackpack.Add(entity, &gc.LocationInBackpack{})
	}
	if comp.StatsChanged != nil {
		c.StatsChanged.Add(entity, &gc.StatsChanged{})
	}

	// Stackable (Countフィールドあり)
	if comp.Stackable != nil {
		stackable := gc.Stackable{}
		if count, ok := (*comp.Stackable)["Count"]; ok {
			if countFloat, ok := count.(float64); ok {
				stackable.Count = int(countFloat)
			}
		}
		c.Stackable.Add(entity, &stackable)
	}

	// データコンポーネント
	restoreDataComponents(entity, comp, c)

	// アイテム属性コンポーネント
	restoreItemComponents(entity, comp, c)

	// アイテム効果コンポーネント
	restoreEffectComponents(entity, comp, c)

	// 戦闘コンポーネント
	if comp.CommandTable != nil {
		c.CommandTable.Add(entity, &gc.CommandTable{Name: comp.CommandTable.Name})
	}

	// 隊員コンポーネント
	if comp.SquadMember != nil {
		c.SquadMember.Add(entity, &gc.SquadMember{})
	}
	if comp.SquadPolicy != nil {
		aiFromSaveData(entity, c, *comp.SquadPolicy)
	}
	// LocationEquipped (Owner以外を復元。Ownerは後で解決)
	if comp.LocationEquipped != nil {
		slot := gc.EquipmentSlotNumber(comp.LocationEquipped.EquipmentSlot)
		if slot >= gc.SlotHead && slot <= gc.SlotWeapon5 {
			equipped := gc.LocationEquipped{
				Owner:         0,
				EquipmentSlot: slot,
			}
			c.LocationEquipped.Add(entity, &equipped)
		}
	}
}

// restoreDataComponents はデータ/表示コンポーネントを復元する
func restoreDataComponents(entity ecs.Entity, comp oapi.SaveDataComponentsMap, c *gc.Components) {
	if comp.Name != nil {
		c.Name.Add(entity, &gc.Name{Name: comp.Name.Name})
	}
	if comp.Description != nil {
		c.Description.Add(entity, &gc.Description{Description: comp.Description.Description})
	}
	if comp.HP != nil {
		hp := hpFromSaveData(*comp.HP)
		c.HP.Add(entity, &hp)
	}
	if comp.WeightCapacity != nil {
		cw := weightCapacityFromSaveData(*comp.WeightCapacity)
		c.WeightCapacity.Add(entity, &cw)
	}
	if comp.TurnBased != nil {
		tb := turnBasedFromSaveData(*comp.TurnBased)
		c.TurnBased.Add(entity, &tb)
	}
	if comp.Abilities != nil {
		ab := abilitiesFromSaveData(*comp.Abilities)
		c.Abilities.Add(entity, &ab)
	}
	if comp.HealthStatus != nil {
		hs := healthStatusFromSaveData(*comp.HealthStatus)
		c.HealthStatus.Add(entity, &hs)
	}
	if comp.Skills != nil {
		skills := skillsFromSaveData(*comp.Skills)
		c.Skills.Add(entity, skills)
	}
	if comp.Camera != nil {
		cam := cameraFromSaveData(*comp.Camera)
		c.Camera.Add(entity, &cam)
	}
	if comp.GridElement != nil {
		ge := gridElementFromSaveData(*comp.GridElement)
		c.GridElement.Add(entity, &ge)
	}
	if comp.SpriteRender != nil {
		sr := spriteRenderFromSaveData(*comp.SpriteRender)
		c.SpriteRender.Add(entity, &sr)
	}
	if comp.LightSource != nil {
		ls := lightSourceFromSaveData(*comp.LightSource)
		c.LightSource.Add(entity, &ls)
	}
}

// restoreItemComponents はアイテム属性コンポーネントを復元する
func restoreItemComponents(entity ecs.Entity, comp oapi.SaveDataComponentsMap, c *gc.Components) {
	if comp.Wearable != nil {
		w := wearableFromSaveData(*comp.Wearable)
		c.Wearable.Add(entity, &w)
	}
	if comp.Value != nil {
		c.Value.Add(entity, &gc.Value{Value: int(comp.Value.Value)})
	}
	if comp.Melee != nil {
		m := meleeFromSaveData(*comp.Melee)
		c.Melee.Add(entity, &m)
	}
	if comp.Fire != nil {
		f := fireFromSaveData(*comp.Fire)
		c.Fire.Add(entity, &f)
	}
	if comp.Recipe != nil {
		r := recipeFromSaveData(*comp.Recipe)
		c.Recipe.Add(entity, &r)
	}
	if comp.Ammo != nil {
		a := ammoFromSaveData(*comp.Ammo)
		c.Ammo.Add(entity, &a)
	}
}

// restoreEffectComponents はアイテム効果コンポーネントを復元する
func restoreEffectComponents(entity ecs.Entity, comp oapi.SaveDataComponentsMap, c *gc.Components) {
	if comp.Consumable != nil {
		con := consumableFromSaveData(*comp.Consumable)
		c.Consumable.Add(entity, &con)
	}
	if comp.ProvidesHealing != nil {
		ph := providesHealingFromSaveData(*comp.ProvidesHealing)
		c.ProvidesHealing.Add(entity, &ph)
	}
	if comp.ProvidesNutrition != nil {
		c.ProvidesNutrition.Add(entity, &gc.ProvidesNutrition{Amount: int(comp.ProvidesNutrition.Amount)})
	}
	if comp.InflictsDamage != nil {
		c.InflictsDamage.Add(entity, &gc.InflictsDamage{Amount: int(comp.InflictsDamage.Amount)})
	}
	if comp.Wallet != nil {
		c.Wallet.Add(entity, &gc.Wallet{Currency: int(comp.Wallet.Currency)})
	}
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

// calculateChecksum はセーブデータからチェックサムを計算する。
// Checksum自身を除いた全データのJSON表現をSHA-256でハッシュする。
func (sm *SerializationManager) calculateChecksum(data *oapi.SaveDataSaveData) (string, error) {
	// チェックサムフィールドを空にしたコピーでハッシュ計算
	hashTarget := oapi.SaveDataSaveData{
		Version:   data.Version,
		Timestamp: data.Timestamp,
		World:     data.World,
	}
	jsonBytes, err := json.Marshal(hashTarget)
	if err != nil {
		return "", fmt.Errorf("failed to marshal save data for checksum: %w", err)
	}
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
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

	sort.Slice(valid, func(i, j int) bool {
		return timestamps[valid[i]].After(timestamps[valid[j]])
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
		return fmt.Errorf("オートセーブに失敗: %w", err)
	}
	if err := sm.rotateAutoSaves(); err != nil {
		return fmt.Errorf("古いオートセーブの削除に失敗: %w", err)
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
// セーブデータ全体をデシリアライズせず、エンティティのName.Nameフィールドだけを抽出する。
func (sm *SerializationManager) GetSavePlayerName(slotName string) (string, error) {
	data, err := sm.loadDataImpl(slotName)
	if err != nil {
		return "", err
	}
	var partial struct {
		World struct {
			Entities []struct {
				Components struct {
					Player *json.RawMessage `json:"player"`
					Name   *struct {
						Name string `json:"name"`
					} `json:"name"`
				} `json:"components"`
			} `json:"entities"`
		} `json:"world"`
	}
	if err := json.Unmarshal(data, &partial); err != nil {
		return "", fmt.Errorf("failed to parse save data: %w", err)
	}
	for _, entity := range partial.World.Entities {
		if entity.Components.Player != nil && entity.Components.Name != nil {
			return entity.Components.Name.Name, nil
		}
	}
	return "", fmt.Errorf("player entity not found in save data")
}

// validateChecksum はセーブデータのチェックサムを検証する
func (sm *SerializationManager) validateChecksum(data *oapi.SaveDataSaveData) error {
	if data.Checksum == "" {
		return fmt.Errorf("checksum field is missing: このセーブデータは改ざんされているか、古いバージョンです")
	}
	expected, err := sm.calculateChecksum(data)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	if data.Checksum != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s (データが改ざんされている可能性があります)",
			expected, data.Checksum)
	}
	return nil
}
