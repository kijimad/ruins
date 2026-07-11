package components

import (
	"fmt"
	"image/color"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/mlange-42/ark/ecs"
)

// EntitySpec はエンティティ作成用の仕様定義
// エンティティに付与するコンポーネントのセットを定義し、
// AddEntities関数でECSエンティティに変換される
type EntitySpec struct {
	// general ================
	Name        *Name
	Description *Description

	// item ================
	HP              *HP
	Consumable      *Consumable
	WeightCapacity  *WeightCapacity
	Melee           *Melee
	Fire            *Fire
	Value           *Value
	Weight          *Weight
	Recipe          *Recipe
	Wearable        *Wearable
	Abilities       *Abilities
	Ammo            *Ammo
	Stackable       *Stackable
	Material        *Material
	LocationOnField *LocationOnField

	// field ================
	Tile            *Tile
	SoloAI          *SoloAI
	SquadAI         *SquadAI
	Camera          *Camera
	Position        *Position
	GridElement     *GridElement
	SpriteRender    *SpriteRender
	BlockView       *BlockView
	BlockPass       *BlockPass
	PassCost        *PassCost
	TurnBased       *TurnBased
	Prop            *Prop
	LightSource     *LightSource
	Door            *Door
	Interactable    *Interactable
	VisualEffect    *VisualEffects
	TileTemperature *TileTemperature

	// member ================
	Player        *Player
	Profession    *Profession
	Hunger        *Hunger
	Wallet        *Wallet
	Faction       *Faction
	Dead          *Dead
	Dialog        *Dialog
	HealthStatus  *HealthStatus
	Skills        *Skills
	CharModifiers *CharModifiers

	// event ================
	StatsChanged      *StatsChanged
	ProvidesHealing   *ProvidesHealing
	ProvidesNutrition *ProvidesNutrition
	InflictsDamage    *InflictsDamage

	// book ================
	Book *Book

	// battle ================
	CommandTable *CommandTable
	DropTable    *DropTable

	// squad ================
	SquadMember *SquadMember

	// singleton ================
	GameLog *GameLog
}

// Components はECSコンポーネントのハンドル束。
// 各コンポーネント型の型付き *ecs.Map[T] を保持し、Add/Has/Get やクエリに使用される。
// コンポーネント追加時は InitializeComponents と本構造体の両方を更新する
type Components struct {
	// general ================
	Name        *ecs.Map[Name]
	Description *ecs.Map[Description]

	// item ================
	HP                 *ecs.Map[HP]
	Consumable         *ecs.Map[Consumable]
	WeightCapacity     *ecs.Map[WeightCapacity]
	Melee              *ecs.Map[Melee]
	Fire               *ecs.Map[Fire]
	Value              *ecs.Map[Value]
	Weight             *ecs.Map[Weight]
	Recipe             *ecs.Map[Recipe]
	Wearable           *ecs.Map[Wearable]
	Abilities          *ecs.Map[Abilities]
	Ammo               *ecs.Map[Ammo]
	Stackable          *ecs.Map[Stackable]
	Material           *ecs.Map[Material]
	LocationInBackpack *ecs.Map[LocationInBackpack]
	LocationEquipped   *ecs.Map[LocationEquipped]
	LocationOnField    *ecs.Map[LocationOnField]
	LocationInStorage  *ecs.Map[LocationInStorage]

	// field ================
	Tile            *ecs.Map[Tile]
	SoloAI          *ecs.Map[SoloAI]
	SquadAI         *ecs.Map[SquadAI]
	Camera          *ecs.Map[Camera]
	Position        *ecs.Map[Position]
	GridElement     *ecs.Map[GridElement]
	SpriteRender    *ecs.Map[SpriteRender]
	BlockView       *ecs.Map[BlockView]
	BlockPass       *ecs.Map[BlockPass]
	PassCost        *ecs.Map[PassCost]
	Door            *ecs.Map[Door]
	Prop            *ecs.Map[Prop]
	LightSource     *ecs.Map[LightSource]
	Interactable    *ecs.Map[Interactable]
	VisualEffect    *ecs.Map[VisualEffects]
	TileTemperature *ecs.Map[TileTemperature]

	// member ================
	Player        *ecs.Map[Player]
	Profession    *ecs.Map[Profession]
	Hunger        *ecs.Map[Hunger]
	Wallet        *ecs.Map[Wallet]
	Faction       *ecs.Map[Faction]
	Boss          *ecs.Map[Boss] // ボスエンティティのマーカー
	Dialog        *ecs.Map[Dialog]
	Dead          *ecs.Map[Dead]
	TurnBased     *ecs.Map[TurnBased]
	HealthStatus  *ecs.Map[HealthStatus]
	Skills        *ecs.Map[Skills]
	CharModifiers *ecs.Map[CharModifiers]

	// event ================
	StateChangeRequest *ecs.Map[StateChangeRequest] // ステート遷移リクエスト
	StatsChanged       *ecs.Map[StatsChanged]
	WeightDirty        *ecs.Map[WeightDirty]
	ProvidesHealing    *ecs.Map[ProvidesHealing]
	ProvidesNutrition  *ecs.Map[ProvidesNutrition]
	InflictsDamage     *ecs.Map[InflictsDamage]

	// book ================
	Book *ecs.Map[Book]

	// battle ================
	CommandTable *ecs.Map[CommandTable]
	DropTable    *ecs.Map[DropTable]

	// squad ================
	SquadMember *ecs.Map[SquadMember]

	// activity ================
	Activity     *ecs.Map[Activity]     // 実行中のアクティビティ
	LastActivity *ecs.Map[LastActivity] // 直近のアクティビティ実行結果

	// singleton ================
	GameLog      *ecs.Map[GameLog]      // フィールドログストレージ
	DungeonState *ecs.Map[Dungeon]      // ダンジョン状態
	GameProgress *ecs.Map[GameProgress] // ゲーム進行データ
	TurnState    *ecs.Map[TurnState]    // ターン状態
	SpatialIndex *ecs.Map[SpatialIndex] // 空間インデックス
}

// Upsert はコンポーネントを追加または更新する。
// Arkの Add は既存でパニックし、Set は不在でパニックするため、Has判定で使い分ける。
// 死亡エンティティには設定できずエラーを返す（ArkのHas/Add/Setは死亡でパニックするため事前に弾く）。
func Upsert[T any](world *ecs.World, comp *ecs.Map[T], entity ecs.Entity, data *T) error {
	if !world.Alive(entity) {
		return fmt.Errorf("死亡エンティティにコンポーネントを設定できない: entity=%v", entity)
	}
	if comp.Has(entity) {
		comp.Set(entity, data)
	} else {
		comp.Add(entity, data)
	}
	return nil
}

// addComp は非nilなら対応するMapにコンポーネントを付与する
func addComp[T any](m *ecs.Map[T], entity ecs.Entity, v *T) {
	if v != nil {
		m.Add(entity, v)
	}
}

// AddEntity は EntitySpec から新しいエンティティを生成し、非nilな各フィールドを
// 対応するコンポーネントとして付与する。リフレクションを使わず型安全に列挙する。
// EntitySpec にフィールドを追加したら本メソッドも更新すること
// （TestAddEntity_AllFields が全フィールドの網羅を検証する）。
func (c *Components) AddEntity(world *ecs.World, spec *EntitySpec) ecs.Entity {
	entity := world.NewEntity()

	addComp(c.Name, entity, spec.Name)
	addComp(c.Description, entity, spec.Description)
	addComp(c.HP, entity, spec.HP)
	addComp(c.Consumable, entity, spec.Consumable)
	addComp(c.WeightCapacity, entity, spec.WeightCapacity)
	addComp(c.Melee, entity, spec.Melee)
	addComp(c.Fire, entity, spec.Fire)
	addComp(c.Value, entity, spec.Value)
	addComp(c.Weight, entity, spec.Weight)
	addComp(c.Recipe, entity, spec.Recipe)
	addComp(c.Wearable, entity, spec.Wearable)
	addComp(c.Abilities, entity, spec.Abilities)
	addComp(c.Ammo, entity, spec.Ammo)
	addComp(c.Stackable, entity, spec.Stackable)
	addComp(c.Material, entity, spec.Material)
	addComp(c.Tile, entity, spec.Tile)
	addComp(c.SoloAI, entity, spec.SoloAI)
	addComp(c.SquadAI, entity, spec.SquadAI)
	addComp(c.Camera, entity, spec.Camera)
	addComp(c.Position, entity, spec.Position)
	addComp(c.GridElement, entity, spec.GridElement)
	addComp(c.SpriteRender, entity, spec.SpriteRender)
	addComp(c.BlockView, entity, spec.BlockView)
	addComp(c.BlockPass, entity, spec.BlockPass)
	addComp(c.PassCost, entity, spec.PassCost)
	addComp(c.TurnBased, entity, spec.TurnBased)
	addComp(c.Prop, entity, spec.Prop)
	addComp(c.LightSource, entity, spec.LightSource)
	addComp(c.Door, entity, spec.Door)
	addComp(c.Interactable, entity, spec.Interactable)
	addComp(c.VisualEffect, entity, spec.VisualEffect)
	addComp(c.TileTemperature, entity, spec.TileTemperature)
	addComp(c.Player, entity, spec.Player)
	addComp(c.Profession, entity, spec.Profession)
	addComp(c.Hunger, entity, spec.Hunger)
	addComp(c.Wallet, entity, spec.Wallet)
	addComp(c.Dead, entity, spec.Dead)
	addComp(c.Dialog, entity, spec.Dialog)
	addComp(c.HealthStatus, entity, spec.HealthStatus)
	addComp(c.Skills, entity, spec.Skills)
	addComp(c.CharModifiers, entity, spec.CharModifiers)
	addComp(c.StatsChanged, entity, spec.StatsChanged)
	addComp(c.ProvidesHealing, entity, spec.ProvidesHealing)
	addComp(c.ProvidesNutrition, entity, spec.ProvidesNutrition)
	addComp(c.InflictsDamage, entity, spec.InflictsDamage)
	addComp(c.Book, entity, spec.Book)
	addComp(c.CommandTable, entity, spec.CommandTable)
	addComp(c.DropTable, entity, spec.DropTable)
	addComp(c.SquadMember, entity, spec.SquadMember)
	addComp(c.GameLog, entity, spec.GameLog)
	addComp(c.Faction, entity, spec.Faction)
	// 位置は生成時にはフィールド配置のみ。backpack/equipped/storage は MoveToX 経由で設定する
	addComp(c.LocationOnField, entity, spec.LocationOnField)

	return entity
}

// InitializeComponents は全コンポーネント型を Ark のワールドに登録し、
// 各フィールドに型付き Map ハンドルを割り当てる。
// Ark は generics で型を実体化するためリフレクションは使えず、明示的に列挙する。
// コンポーネント追加時はこの関数と Components 構造体の両方を更新する。
func (c *Components) InitializeComponents(world *ecs.World) error {
	c.Name = ecs.NewMap[Name](world)
	c.Description = ecs.NewMap[Description](world)
	c.HP = ecs.NewMap[HP](world)
	c.Consumable = ecs.NewMap[Consumable](world)
	c.WeightCapacity = ecs.NewMap[WeightCapacity](world)
	c.Melee = ecs.NewMap[Melee](world)
	c.Fire = ecs.NewMap[Fire](world)
	c.Value = ecs.NewMap[Value](world)
	c.Weight = ecs.NewMap[Weight](world)
	c.Recipe = ecs.NewMap[Recipe](world)
	c.Wearable = ecs.NewMap[Wearable](world)
	c.Abilities = ecs.NewMap[Abilities](world)
	c.Ammo = ecs.NewMap[Ammo](world)
	c.Stackable = ecs.NewMap[Stackable](world)
	c.Material = ecs.NewMap[Material](world)
	c.LocationInBackpack = ecs.NewMap[LocationInBackpack](world)
	c.LocationEquipped = ecs.NewMap[LocationEquipped](world)
	c.LocationOnField = ecs.NewMap[LocationOnField](world)
	c.LocationInStorage = ecs.NewMap[LocationInStorage](world)
	c.Tile = ecs.NewMap[Tile](world)
	c.SoloAI = ecs.NewMap[SoloAI](world)
	c.SquadAI = ecs.NewMap[SquadAI](world)
	c.Camera = ecs.NewMap[Camera](world)
	c.Position = ecs.NewMap[Position](world)
	c.GridElement = ecs.NewMap[GridElement](world)
	c.SpriteRender = ecs.NewMap[SpriteRender](world)
	c.BlockView = ecs.NewMap[BlockView](world)
	c.BlockPass = ecs.NewMap[BlockPass](world)
	c.PassCost = ecs.NewMap[PassCost](world)
	c.Door = ecs.NewMap[Door](world)
	c.Prop = ecs.NewMap[Prop](world)
	c.LightSource = ecs.NewMap[LightSource](world)
	c.Interactable = ecs.NewMap[Interactable](world)
	c.VisualEffect = ecs.NewMap[VisualEffects](world)
	c.TileTemperature = ecs.NewMap[TileTemperature](world)
	c.Player = ecs.NewMap[Player](world)
	c.Profession = ecs.NewMap[Profession](world)
	c.Hunger = ecs.NewMap[Hunger](world)
	c.Wallet = ecs.NewMap[Wallet](world)
	c.Faction = ecs.NewMap[Faction](world)
	c.Boss = ecs.NewMap[Boss](world)
	c.Dialog = ecs.NewMap[Dialog](world)
	c.Dead = ecs.NewMap[Dead](world)
	c.TurnBased = ecs.NewMap[TurnBased](world)
	c.HealthStatus = ecs.NewMap[HealthStatus](world)
	c.Skills = ecs.NewMap[Skills](world)
	c.CharModifiers = ecs.NewMap[CharModifiers](world)
	c.StateChangeRequest = ecs.NewMap[StateChangeRequest](world)
	c.StatsChanged = ecs.NewMap[StatsChanged](world)
	c.WeightDirty = ecs.NewMap[WeightDirty](world)
	c.ProvidesHealing = ecs.NewMap[ProvidesHealing](world)
	c.ProvidesNutrition = ecs.NewMap[ProvidesNutrition](world)
	c.InflictsDamage = ecs.NewMap[InflictsDamage](world)
	c.Book = ecs.NewMap[Book](world)
	c.CommandTable = ecs.NewMap[CommandTable](world)
	c.DropTable = ecs.NewMap[DropTable](world)
	c.SquadMember = ecs.NewMap[SquadMember](world)
	c.Activity = ecs.NewMap[Activity](world)
	c.LastActivity = ecs.NewMap[LastActivity](world)
	c.GameLog = ecs.NewMap[GameLog](world)
	c.DungeonState = ecs.NewMap[Dungeon](world)
	c.GameProgress = ecs.NewMap[GameProgress](world)
	c.TurnState = ecs.NewMap[TurnState](world)
	c.SpatialIndex = ecs.NewMap[SpatialIndex](world)
	return nil
}

// Camera はカメラ
// 滑らかなズームと追従のため、実際値と目標値を別々に持つ
type Camera struct {
	// ズーム率
	Scale   float64
	ScaleTo float64
	// カメラ位置。ピクセル単位
	Pos    consts.Coord[float64]
	Target consts.Coord[float64]
}

// Consumable は消耗品。一度使うとなくなる
type Consumable struct {
	UsableScene UsableSceneType
	TargetType  TargetType
}

// Name は表示名
type Name struct {
	Name string
}

// Description は説明
type Description struct {
	Description string
}

// Dialog は会話データ
type Dialog struct {
	MessageKey string // メッセージキー
}

// Wearable は装備品。キャラクタが装備することでパラメータを変更できる
type Wearable struct {
	Defense           int           // 防御力
	EquipmentCategory EquipmentType // 装備部位
	EquipBonus        EquipBonus    // ステータスへのボーナス
	InsulationCold    int           // 耐寒（快適温度の下限を下げる）
	InsulationHeat    int           // 耐暑（快適温度の上限を上げる）
}

// Player は操作対象の主人公キャラクター
type Player struct{}

// Boss はボスエンティティを示すマーカーコンポーネント
type Boss struct{}

// Profession はプレイヤーが選択した職業を保持する。ラン終了時の再適用に使う
type Profession struct {
	ID string // raw.Profession.ID に対応する
}

// Dead はキャラクターが死亡している状態を示すマーカーコンポーネント
// 死亡時の処理(ドロップ/統計処理/ゲームログ...)を共通化するために使う
type Dead struct{}

// Wallet はプレイヤーの資金を管理する
type Wallet struct {
	Currency int
}

// HP は生命力を表すコンポーネント
// なくなるとゲームオーバーになる。キャラクターとProp（破壊可能な置物）の両方が使う
type HP Pool[int]

// WeightCapacity は重量容量を表すコンポーネント。
// Playerの所持重量とStorageの格納重量の両方に使用する。
// Maxは最大容量、Currentは現在の重量を表す
type WeightCapacity Pool[float64]

// ProvidesHealing は回復する性質。
// Amount の意味は Kind で決まる。HealNumeral なら絶対回復量、HealRatio なら最大HPに対する倍率。
type ProvidesHealing struct {
	Kind   HealAmountKind
	Amount float64
}

// Calc は基準値(最大HP)から実際の回復量を計算する。絶対量指定の場合baseは無視される
func (ph ProvidesHealing) Calc(base int) int {
	if ph.Kind == HealRatio {
		return int(float64(base) * ph.Amount)
	}
	return int(ph.Amount)
}

// ProvidesNutrition は空腹度を回復する性質
type ProvidesNutrition struct {
	Amount int // 回復量（この値だけ空腹度を減らす）
}

// InflictsDamage はダメージを与える性質
// 直接的な数値が作用し、ステータスなどは考慮されない
type InflictsDamage struct {
	Amount int
}

// Stackable はスタック可能なエンティティを示すコンポーネント
// 所持数を管理する。非Stackableエンティティの個数は常に1として扱う
type Stackable struct {
	Count int // 所持数
}

// Value はアイテムの基本価値
// 売買時の基準となる。実際の売値・買値は店や状況に応じて倍率が適用される
type Value struct {
	Value int
}

// Weight はアイテムの重量(kg)
// 所持重量の計算に使用される
type Weight struct {
	Kg float64 // 重量（キログラム）
}

// Recipe は合成に必要な素材
type Recipe struct {
	Inputs []RecipeInput
}

// StatsChanged はステータス再計算が必要なことを示すダーティーフラグ
// フラグ系コンポーネントは、トリガーした順序に関わらず安定して実行させるために使う
type StatsChanged struct{}

// WeightDirty は重量の再計算が必要であることを示すダーティフラグ
// フラグ系コンポーネントは、トリガーした順序に関わらず安定して実行させるために使う
type WeightDirty struct{}

// Ammo は弾薬アイテムの性能を定義する
type Ammo struct {
	AmmoTag       string // 口径タグ。武器の AmmoTag とマッチする
	DamageBonus   int    // ダメージ修正値
	AccuracyBonus int    // 命中率修正値
}

// Attacker は近接・遠距離攻撃の共通インターフェース。
// ダメージ計算や命中判定など攻撃種別を問わない共通処理で使用する
type Attacker interface {
	GetAccuracy() int
	GetDamage() int
	GetAttackCount() int
	GetElement() ElementType
	GetAttackCategory() AttackType
	GetCost() int
	GetTargetType() TargetType
}

// Melee は近接攻撃の性質。近接攻撃毎にこの数値と作用対象のステータスを加味して、最終的なダメージ量を決定する
type Melee struct {
	Accuracy       int         // 命中率
	Damage         int         // 攻撃力
	AttackCount    int         // 攻撃回数
	Element        ElementType // 攻撃属性
	AttackCategory AttackType  // 攻撃種別
	Cost           int         // 行動コスト
	TargetType     TargetType  // 対象タイプ
}

// GetAccuracy はAttackerの実装
func (m *Melee) GetAccuracy() int { return m.Accuracy }

// GetDamage はAttackerの実装
func (m *Melee) GetDamage() int { return m.Damage }

// GetAttackCount はAttackerの実装
func (m *Melee) GetAttackCount() int { return m.AttackCount }

// GetElement はAttackerの実装
func (m *Melee) GetElement() ElementType { return m.Element }

// GetAttackCategory はAttackerの実装
func (m *Melee) GetAttackCategory() AttackType { return m.AttackCategory }

// GetCost はAttackerの実装
func (m *Melee) GetCost() int { return m.Cost }

// GetTargetType はAttackerの実装
func (m *Melee) GetTargetType() TargetType { return m.TargetType }

// Fire は遠距離攻撃の性質。射撃パラメータと弾薬管理を含む
type Fire struct {
	// 攻撃パラメータ
	Accuracy       int         // 命中率
	Damage         int         // 攻撃力
	AttackCount    int         // 攻撃回数
	Element        ElementType // 攻撃属性
	AttackCategory AttackType  // 攻撃種別
	Cost           int         // 行動コスト
	TargetType     TargetType  // 対象タイプ
	// 弾薬管理
	Magazine            int    // 現在の装弾数
	MagazineSize        int    // 最大装弾数
	ReloadEffort        int    // リロード完了に必要な総工数
	AmmoTag             string // 使用する弾薬の口径タグ。Ammoコンポーネントの AmmoTag とマッチする
	LoadedDamageBonus   int    // 装填中の弾薬によるダメージ修正値。リロード時に設定される
	LoadedAccuracyBonus int    // 装填中の弾薬による命中修正値。リロード時に設定される
}

// GetAccuracy はAttackerの実装
func (f *Fire) GetAccuracy() int { return f.Accuracy }

// GetDamage はAttackerの実装
func (f *Fire) GetDamage() int { return f.Damage }

// GetAttackCount はAttackerの実装
func (f *Fire) GetAttackCount() int { return f.AttackCount }

// GetElement はAttackerの実装
func (f *Fire) GetElement() ElementType { return f.Element }

// GetAttackCategory はAttackerの実装
func (f *Fire) GetAttackCategory() AttackType { return f.AttackCategory }

// GetCost はAttackerの実装
func (f *Fire) GetCost() int { return f.Cost }

// GetTargetType はAttackerの実装
func (f *Fire) GetTargetType() TargetType { return f.TargetType }

// CommandTable はAI用の、戦闘コマンドテーブル名
type CommandTable struct {
	Name string
}

// DropTable はドロップテーブル名
type DropTable struct {
	Name string
}

// SheetImage はシート画像情報
type SheetImage struct {
	SheetName   string
	SheetNumber *int
}

// FactionKind は所属派閥の種別。絶対的な指定
type FactionKind int

const (
	// FactionAlly は味方(プレイヤー側)
	FactionAlly FactionKind = iota
	// FactionEnemy は敵性(プレイヤーと敵対)
	FactionEnemy
	// FactionNeutral は中立(会話可能NPC)
	FactionNeutral
)

// String は派閥種別の文字列表現を返す。oapiのenum文字列と一致させる
func (k FactionKind) String() string {
	switch k {
	case FactionAlly:
		return "FactionAlly"
	case FactionEnemy:
		return "FactionEnemy"
	case FactionNeutral:
		return "FactionNeutral"
	default:
		return "Unknown"
	}
}

// Faction は所属派閥を表すコンポーネント。Kindで種別を判別するタグ付きデータ
type Faction struct {
	Kind FactionKind
}

// 位置は種別ごとに独立したコンポーネントとして表現する。
// archetypeクエリ（「全装備品」「全バックパックアイテム」）が効くため、
// タグ付き単一コンポーネントより archetype ECS に適している。
// 排他（1エンティティは高々1つの位置）は lifecycle の MoveToX 関数で保証する。

// LocationInBackpack はバックパック内位置
type LocationInBackpack struct {
	Owner ecs.Entity // バックパックの所有者
}

// LocationEquipped は装備中位置
type LocationEquipped struct {
	Owner         ecs.Entity
	EquipmentSlot EquipmentSlotNumber
}

// LocationOnField はフィールド上位置
type LocationOnField struct{}

// LocationInStorage は収納内位置
type LocationInStorage struct {
	Owner ecs.Entity // 収納Propのエンティティ
}

// Material は素材を表すマーカーコンポーネント。
// 合成や売却の材料となるアイテムに付与される
type Material struct{}

// Prop は置物を表すマーカーコンポーネント
type Prop struct{}

// LightSource は光源コンポーネント
type LightSource struct {
	Radius  consts.Tile // 照明範囲
	Color   color.RGBA  // 光の色
	Enabled bool        // 有効/無効
}

// Door は開閉可能な扉コンポーネント
type Door struct {
	IsOpen      bool            // 開いているかどうか
	Orientation DoorOrientation // 扉の向き
	Locked      bool            // ロック中は開閉不可
}

// DoorOrientation は扉の向き
type DoorOrientation int

const (
	// DoorOrientationHorizontal は横向きの扉
	DoorOrientationHorizontal DoorOrientation = iota
	// DoorOrientationVertical は縦向きの扉
	DoorOrientationVertical
)
