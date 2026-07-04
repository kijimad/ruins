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
	ecs "github.com/x-hgg-x/goecs/v2"
)

const saveDataVersion = "1.0.0"

const maxAutoSaves = 4

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
func NewSerializationManager(opts ...Option) *SerializationManager {
	sm := &SerializationManager{
		saveDirectory:   defaultSaveDir,
		stableIDManager: NewStableIDManager(),
	}
	for _, opt := range opts {
		opt(sm)
	}
	sm.initImpl()
	return sm
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
	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(collect))
	// 2. バックパック内アイテム
	world.Manager.Join(world.Components.LocationInBackpack).Visit(ecs.Visit(collect))
	// 3. 装備中アイテム
	world.Manager.Join(world.Components.LocationEquipped).Visit(ecs.Visit(collect))
	// 4. 隊員エンティティ（死亡していないもの）
	world.Manager.Join(world.Components.SquadMember).Visit(ecs.Visit(func(entity ecs.Entity) {
		if !entity.HasComponent(world.Components.Dead) {
			collect(entity)
		}
	}))

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
	if entity.HasComponent(c.Player) {
		comp.Player = emptyMarker()
	}
	if entity.HasComponent(c.FactionAlly) {
		comp.FactionAllyData = emptyMarker()
	}
	if entity.HasComponent(c.LocationInBackpack) {
		backpack := c.LocationInBackpack.Get(entity).(*gc.LocationInBackpack)
		ownerStableID := sm.stableIDManager.GetStableID(backpack.Owner)
		comp.LocationInBackpack = &oapi.SaveDataLocationInBackpackComponent{
			OwnerRef: stableIDToSaveData(ownerStableID),
		}
	}
	if entity.HasComponent(c.StatsChanged) {
		comp.StatsChanged = emptyMarker()
	}
	if entity.HasComponent(c.Stackable) {
		stackable := c.Stackable.Get(entity).(*gc.Stackable)
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
	if entity.HasComponent(c.Name) {
		name := c.Name.Get(entity).(*gc.Name)
		comp.Name = &oapi.SaveDataNameComponent{Name: name.Name}
	}
	if entity.HasComponent(c.Description) {
		desc := c.Description.Get(entity).(*gc.Description)
		comp.Description = &oapi.SaveDataDescriptionComponent{Description: desc.Description}
	}
	if entity.HasComponent(c.HP) {
		sd := hpToSaveData(*c.HP.Get(entity).(*gc.HP))
		comp.HP = &sd
	}
	if entity.HasComponent(c.WeightCapacity) {
		sd := weightCapacityToSaveData(*c.WeightCapacity.Get(entity).(*gc.WeightCapacity))
		comp.WeightCapacity = &sd
	}
	if entity.HasComponent(c.TurnBased) {
		tb := c.TurnBased.Get(entity).(*gc.TurnBased)
		sd := turnBasedToSaveData(*tb)
		comp.TurnBased = &sd
	}
	if entity.HasComponent(c.Abilities) {
		ab := c.Abilities.Get(entity).(*gc.Abilities)
		sd := abilitiesToSaveData(*ab)
		comp.Abilities = &sd
	}

	// 表示コンポーネント
	if entity.HasComponent(c.Camera) {
		cam := c.Camera.Get(entity).(*gc.Camera)
		sd := cameraToSaveData(*cam)
		comp.Camera = &sd
	}
	if entity.HasComponent(c.GridElement) {
		ge := c.GridElement.Get(entity).(*gc.GridElement)
		sd := gridElementToSaveData(*ge)
		comp.GridElement = &sd
	}
	if entity.HasComponent(c.SpriteRender) {
		sr := c.SpriteRender.Get(entity).(*gc.SpriteRender)
		sd := spriteRenderToSaveData(*sr)
		comp.SpriteRender = &sd
	}
	if entity.HasComponent(c.LightSource) {
		ls := c.LightSource.Get(entity).(*gc.LightSource)
		sd := lightSourceToSaveData(*ls)
		comp.LightSource = &sd
	}

	// アイテム属性コンポーネント
	if entity.HasComponent(c.Wearable) {
		w := c.Wearable.Get(entity).(*gc.Wearable)
		sd := wearableToSaveData(*w)
		comp.Wearable = &sd
	}
	if entity.HasComponent(c.Value) {
		v := c.Value.Get(entity).(*gc.Value)
		comp.Value = &oapi.SaveDataValueComponent{Value: int32(v.Value)}
	}
	if entity.HasComponent(c.Melee) {
		m := c.Melee.Get(entity).(*gc.Melee)
		sd := meleeToSaveData(*m)
		comp.Melee = &sd
	}
	if entity.HasComponent(c.Fire) {
		f := c.Fire.Get(entity).(*gc.Fire)
		sd := fireToSaveData(*f)
		comp.Fire = &sd
	}
	if entity.HasComponent(c.Recipe) {
		r := c.Recipe.Get(entity).(*gc.Recipe)
		sd := recipeToSaveData(*r)
		comp.Recipe = &sd
	}
	if entity.HasComponent(c.Ammo) {
		a := c.Ammo.Get(entity).(*gc.Ammo)
		sd := ammoToSaveData(*a)
		comp.Ammo = &sd
	}

	// アイテム効果コンポーネント
	if entity.HasComponent(c.Consumable) {
		con := c.Consumable.Get(entity).(*gc.Consumable)
		sd := consumableToSaveData(*con)
		comp.Consumable = &sd
	}
	if entity.HasComponent(c.ProvidesHealing) {
		ph := c.ProvidesHealing.Get(entity).(*gc.ProvidesHealing)
		sd := providesHealingToSaveData(*ph)
		comp.ProvidesHealing = &sd
	}
	if entity.HasComponent(c.ProvidesNutrition) {
		pn := c.ProvidesNutrition.Get(entity).(*gc.ProvidesNutrition)
		comp.ProvidesNutrition = &oapi.SaveDataProvidesNutritionComponent{Amount: int32(pn.Amount)}
	}
	if entity.HasComponent(c.InflictsDamage) {
		id := c.InflictsDamage.Get(entity).(*gc.InflictsDamage)
		comp.InflictsDamage = &oapi.SaveDataInflictsDamageComponent{Amount: int32(id.Amount)}
	}
	if entity.HasComponent(c.Wallet) {
		wal := c.Wallet.Get(entity).(*gc.Wallet)
		comp.Wallet = &oapi.SaveDataWalletComponent{Currency: int32(wal.Currency)}
	}

	// 戦闘コンポーネント
	if entity.HasComponent(c.CommandTable) {
		ct := c.CommandTable.Get(entity).(*gc.CommandTable)
		comp.CommandTable = &oapi.SaveDataCommandTableComponent{Name: ct.Name}
	}

	// 隊員コンポーネント
	if entity.HasComponent(c.SquadMember) {
		comp.SquadMember = &oapi.SaveDataSquadMemberComponent{}
	}
	if entity.HasComponent(c.AI) {
		ai := c.AI.Get(entity).(*gc.AI)
		sd := aiToSaveData(*ai)
		comp.SquadPolicy = &sd
	}
	// エンティティ参照コンポーネント (LocationEquipped)
	if entity.HasComponent(c.LocationEquipped) {
		equipped := c.LocationEquipped.Get(entity).(*gc.LocationEquipped)
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

// entityEntry はエンティティ復元時の一時データ
type entityEntry struct {
	entity ecs.Entity
	data   oapi.SaveDataEntitySaveData
}

// restoreWorldData はセーブデータからワールドを復元する
func (sm *SerializationManager) restoreWorldData(world w.World, worldData oapi.SaveDataWorldSaveData) error {
	// 第1段階: 全エンティティを作成して安定IDマッピング
	entries := make([]entityEntry, 0, len(worldData.Entities))
	for _, entityData := range worldData.Entities {
		entity := world.Manager.NewEntity()
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
			backpack := c.LocationInBackpack.Get(entry.entity).(*gc.LocationInBackpack)
			backpack.Owner = ownerEntity
		}

		// LocationEquipped の Owner を解決
		if comp.LocationEquipped != nil {
			ownerStableID := stableIDFromSaveData(comp.LocationEquipped.OwnerRef)
			ownerEntity, exists := sm.stableIDManager.GetEntity(ownerStableID)
			if !exists {
				return fmt.Errorf("required owner entity not found for stable ID: %v", ownerStableID)
			}
			equipped := c.LocationEquipped.Get(entry.entity).(*gc.LocationEquipped)
			equipped.Owner = ownerEntity
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
		entity.AddComponent(c.Player, &gc.Player{})
	}
	if comp.FactionAllyData != nil {
		entity.AddComponent(c.FactionAlly, &gc.FactionAllyData{})
	}
	// LocationInBackpack（Ownerは第3段階で解決）
	if comp.LocationInBackpack != nil {
		entity.AddComponent(c.LocationInBackpack, &gc.LocationInBackpack{})
	}
	if comp.StatsChanged != nil {
		entity.AddComponent(c.StatsChanged, &gc.StatsChanged{})
	}

	// Stackable (Countフィールドあり)
	if comp.Stackable != nil {
		stackable := gc.Stackable{}
		if count, ok := (*comp.Stackable)["Count"]; ok {
			if countFloat, ok := count.(float64); ok {
				stackable.Count = int(countFloat)
			}
		}
		entity.AddComponent(c.Stackable, &stackable)
	}

	// データコンポーネント
	restoreDataComponents(entity, comp, c)

	// アイテム属性コンポーネント
	restoreItemComponents(entity, comp, c)

	// アイテム効果コンポーネント
	restoreEffectComponents(entity, comp, c)

	// 戦闘コンポーネント
	if comp.CommandTable != nil {
		entity.AddComponent(c.CommandTable, &gc.CommandTable{Name: comp.CommandTable.Name})
	}

	// 隊員コンポーネント（Leaderは第3段階で解決）
	if comp.SquadMember != nil {
		entity.AddComponent(c.SquadMember, &gc.SquadMember{})

		entity.AddComponent(c.BlockPass, &gc.BlockPass{})
		entity.AddComponent(c.HealthStatus, &gc.HealthStatus{})
		skills := gc.NewSkills()
		entity.AddComponent(c.Skills, skills)
		if comp.Abilities != nil {
			ab := abilitiesFromSaveData(*comp.Abilities)
			entity.AddComponent(c.CharModifiers, gc.RecalculateCharModifiers(skills, &ab, nil))
		} else {
			entity.AddComponent(c.CharModifiers, gc.RecalculateCharModifiers(skills, nil, nil))
		}
	}
	if comp.SquadPolicy != nil {
		ai := aiFromSaveData(*comp.SquadPolicy)
		entity.AddComponent(c.AI, &ai)
	}
	// LocationEquipped (Owner以外を復元。Ownerは後で解決)
	if comp.LocationEquipped != nil {
		slot := gc.EquipmentSlotNumber(comp.LocationEquipped.EquipmentSlot)
		if slot >= gc.SlotHead && slot <= gc.SlotWeapon5 {
			equipped := gc.LocationEquipped{
				Owner:         0,
				EquipmentSlot: slot,
			}
			entity.AddComponent(c.LocationEquipped, &equipped)
		}
	}
}

// restoreDataComponents はデータ/表示コンポーネントを復元する
func restoreDataComponents(entity ecs.Entity, comp oapi.SaveDataComponentsMap, c *gc.Components) {
	if comp.Name != nil {
		entity.AddComponent(c.Name, &gc.Name{Name: comp.Name.Name})
	}
	if comp.Description != nil {
		entity.AddComponent(c.Description, &gc.Description{Description: comp.Description.Description})
	}
	if comp.HP != nil {
		hp := hpFromSaveData(*comp.HP)
		entity.AddComponent(c.HP, &hp)
	}
	if comp.WeightCapacity != nil {
		cw := weightCapacityFromSaveData(*comp.WeightCapacity)
		entity.AddComponent(c.WeightCapacity, &cw)
	}
	if comp.TurnBased != nil {
		tb := turnBasedFromSaveData(*comp.TurnBased)
		entity.AddComponent(c.TurnBased, &tb)
	}
	if comp.Abilities != nil {
		ab := abilitiesFromSaveData(*comp.Abilities)
		entity.AddComponent(c.Abilities, &ab)
	}
	if comp.Camera != nil {
		cam := cameraFromSaveData(*comp.Camera)
		entity.AddComponent(c.Camera, &cam)
	}
	if comp.GridElement != nil {
		ge := gridElementFromSaveData(*comp.GridElement)
		entity.AddComponent(c.GridElement, &ge)
	}
	if comp.SpriteRender != nil {
		sr := spriteRenderFromSaveData(*comp.SpriteRender)
		entity.AddComponent(c.SpriteRender, &sr)
	}
	if comp.LightSource != nil {
		ls := lightSourceFromSaveData(*comp.LightSource)
		entity.AddComponent(c.LightSource, &ls)
	}
}

// restoreItemComponents はアイテム属性コンポーネントを復元する
func restoreItemComponents(entity ecs.Entity, comp oapi.SaveDataComponentsMap, c *gc.Components) {
	if comp.Wearable != nil {
		w := wearableFromSaveData(*comp.Wearable)
		entity.AddComponent(c.Wearable, &w)
	}
	if comp.Value != nil {
		entity.AddComponent(c.Value, &gc.Value{Value: int(comp.Value.Value)})
	}
	if comp.Melee != nil {
		m := meleeFromSaveData(*comp.Melee)
		entity.AddComponent(c.Melee, &m)
	}
	if comp.Fire != nil {
		f := fireFromSaveData(*comp.Fire)
		entity.AddComponent(c.Fire, &f)
	}
	if comp.Recipe != nil {
		r := recipeFromSaveData(*comp.Recipe)
		entity.AddComponent(c.Recipe, &r)
	}
	if comp.Ammo != nil {
		a := ammoFromSaveData(*comp.Ammo)
		entity.AddComponent(c.Ammo, &a)
	}
}

// restoreEffectComponents はアイテム効果コンポーネントを復元する
func restoreEffectComponents(entity ecs.Entity, comp oapi.SaveDataComponentsMap, c *gc.Components) {
	if comp.Consumable != nil {
		con := consumableFromSaveData(*comp.Consumable)
		entity.AddComponent(c.Consumable, &con)
	}
	if comp.ProvidesHealing != nil {
		ph := providesHealingFromSaveData(*comp.ProvidesHealing)
		entity.AddComponent(c.ProvidesHealing, &ph)
	}
	if comp.ProvidesNutrition != nil {
		entity.AddComponent(c.ProvidesNutrition, &gc.ProvidesNutrition{Amount: int(comp.ProvidesNutrition.Amount)})
	}
	if comp.InflictsDamage != nil {
		entity.AddComponent(c.InflictsDamage, &gc.InflictsDamage{Amount: int(comp.InflictsDamage.Amount)})
	}
	if comp.Wallet != nil {
		entity.AddComponent(c.Wallet, &gc.Wallet{Currency: int(comp.Wallet.Currency)})
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

// RotateAutoSaves はオートセーブを最大件数まで削減する。
// 古い順に削除して maxAutoSaves 件を保持する。
func (sm *SerializationManager) RotateAutoSaves() error {
	saves, err := sm.ListSaves()
	if err != nil {
		return err
	}

	var autoSaves []string
	for _, name := range saves {
		if strings.HasPrefix(name, "auto_") {
			autoSaves = append(autoSaves, name)
		}
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
