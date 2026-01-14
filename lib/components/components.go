package components

import (
	"fmt"
	"image/color"
	"reflect"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// EntitySpec はエンティティ作成用の仕様定義
// エンティティに付与するコンポーネントのセットを定義し、
// AddEntities関数でECSエンティティに変換される
type EntitySpec struct {
	// general ================
	Name        *Name
	Description *Description

	// item ================
	Item             *Item
	Consumable       *Consumable
	Pools            *Pools
	Attack           *Attack
	Value            *Value
	Weight           *Weight
	Recipe           *Recipe
	Wearable         *Wearable
	Attributes       *Attributes
	Weapon           *Weapon
	Stackable        *Stackable
	ItemLocationType *ItemLocationType

	// field ================
	AIMoveFSM    *AIMoveFSM
	AIRoaming    *AIRoaming
	AIVision     *AIVision
	AIChasing    *AIChasing
	Camera       *Camera
	Position     *Position
	GridElement  *GridElement
	SpriteRender *SpriteRender
	BlockView    *BlockView
	BlockPass    *BlockPass
	TurnBased    *TurnBased
	Prop         *Prop
	LightSource  *LightSource
	Door         *Door
	Interactable *Interactable

	// member ================
	Player      *Player
	Hunger      *Hunger
	Wallet      *Wallet
	FactionType *FactionType
	Dead        *Dead
	Dialog      *Dialog

	// event ================
	EquipmentChanged  *EquipmentChanged
	ProvidesHealing   *ProvidesHealing
	ProvidesNutrition *ProvidesNutrition
	InflictsDamage    *InflictsDamage

	// battle ================
	CommandTable *CommandTable
	DropTable    *DropTable
}

// Components はECSコンポーネントストレージ
// 各コンポーネント型のSliceComponent/NullComponentを保持し、
// Manager.Join()でのクエリに使用される
type Components struct {
	// general ================
	Name        *ecs.SliceComponent `save:"true"`
	Description *ecs.SliceComponent `save:"true"`

	// item ================
	Item                   *ecs.NullComponent  `save:"true"`
	Consumable             *ecs.SliceComponent `save:"true"`
	Pools                  *ecs.SliceComponent `save:"true"`
	Attack                 *ecs.SliceComponent `save:"true"`
	Value                  *ecs.SliceComponent `save:"true"`
	Weight                 *ecs.SliceComponent `save:"true"`
	Recipe                 *ecs.SliceComponent `save:"true"`
	Wearable               *ecs.SliceComponent `save:"true"`
	Attributes             *ecs.SliceComponent `save:"true"`
	Weapon                 *ecs.SliceComponent `save:"true"`
	Stackable              *ecs.SliceComponent `save:"true"`
	ItemLocationInBackpack *ecs.NullComponent  `save:"true"`
	ItemLocationEquipped   *ecs.SliceComponent `save:"true"`
	ItemLocationOnField    *ecs.NullComponent

	// field ================
	AIMoveFSM    *ecs.SliceComponent
	AIRoaming    *ecs.SliceComponent
	AIVision     *ecs.SliceComponent
	AIChasing    *ecs.SliceComponent
	Camera       *ecs.SliceComponent
	Position     *ecs.SliceComponent
	GridElement  *ecs.SliceComponent
	SpriteRender *ecs.SliceComponent
	BlockView    *ecs.NullComponent
	BlockPass    *ecs.NullComponent
	Door         *ecs.SliceComponent
	Prop         *ecs.NullComponent
	LightSource  *ecs.SliceComponent `save:"true"`
	Interactable *ecs.SliceComponent

	// member ================
	Player         *ecs.NullComponent `save:"true"`
	Hunger         *ecs.SliceComponent
	Wallet         *ecs.SliceComponent `save:"true"`
	FactionAlly    *ecs.NullComponent  `save:"true"`
	FactionEnemy   *ecs.NullComponent
	FactionNeutral *ecs.NullComponent `save:"true"`
	Dialog         *ecs.SliceComponent
	Dead           *ecs.NullComponent
	TurnBased      *ecs.SliceComponent `save:"true"`

	// event ================
	EquipmentChanged  *ecs.NullComponent
	ProvidesHealing   *ecs.SliceComponent `save:"true"`
	ProvidesNutrition *ecs.SliceComponent `save:"true"`
	InflictsDamage    *ecs.SliceComponent `save:"true"`

	// battle ================
	CommandTable *ecs.SliceComponent
	DropTable    *ecs.SliceComponent
}

// InitializeComponents はComponentInitializerインターフェースを実装する
// リフレクションを使用して自動的に全コンポーネントを初期化する
// コンポーネント追加時の手動更新が不要
func (c *Components) InitializeComponents(manager *ecs.Manager) error {
	val := reflect.ValueOf(c).Elem() // *Components から Components へ
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		fieldName := fieldType.Name

		// フィールドが設定可能かチェック
		if !field.CanSet() {
			return fmt.Errorf("field %s is not settable", fieldName)
		}

		// フィールドの型に基づいて適切なコンポーネントを作成
		switch field.Type() {
		case reflect.TypeOf((*ecs.SliceComponent)(nil)):
			// SliceComponent の初期化
			field.Set(reflect.ValueOf(manager.NewSliceComponent()))
		case reflect.TypeOf((*ecs.NullComponent)(nil)):
			// NullComponent の初期化
			field.Set(reflect.ValueOf(manager.NewNullComponent()))
		default:
			// 未対応の型はエラーとして扱う
			return fmt.Errorf("unsupported component type %v for field %s", field.Type(), fieldName)
		}
	}

	return nil
}

// Camera はカメラ
// 滑らかなズームの変更のため実際のズーム率と対象ズーム率を持つ
type Camera struct {
	Scale   float64
	ScaleTo float64
}

// Item はキャラクターが保持できるもの。フィールド上、装備上、インベントリ上など位置状態を持ち、1スロットを消費する
// 装備品、武器、回復アイテム、売却アイテム、素材など
type Item struct{}

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
}

// Player は操作対象の主人公キャラクター
type Player struct{}

// Dead はキャラクターが死亡している状態を示すマーカーコンポーネント
// 死亡時の処理(ドロップ/統計処理/ゲームログ...)を共通化するために使う
type Dead struct{}

// Wallet はプレイヤーの資金を管理する
type Wallet struct {
	Currency int
}

// Pools はキャラクターのプール情報
type Pools struct {
	// 生命力 Health point
	// なくなるとゲームオーバー
	HP Pool
	// スタミナ Stamina point
	// 走ったり攻撃したら減る。自動回復する
	SP Pool
	// 電力 Electricity point
	// 機能のトグルで消費量が変わる
	EP Pool
	// 所持重量 Weight
	// 超過量に応じたペナルティが発生する
	Weight PoolFloat
}

// Attributes はエンティティが持つステータス値。各種計算式で使う
type Attributes struct {
	Vitality  Attribute // 体力。丈夫さ、持久力、しぶとさ。HPやSPに影響する
	Strength  Attribute // 筋力。主に近接攻撃のダメージに影響する
	Sensation Attribute // 感覚。主に射撃攻撃のダメージに影響する
	Dexterity Attribute // 器用。攻撃時の命中率に影響する
	Agility   Attribute // 敏捷。回避率、行動の速さに影響する
	Defense   Attribute // 防御。被弾ダメージを軽減させる
}

// ProvidesHealing は回復する性質
// 直接的な数値が作用し、ステータスなどは考慮されない
type ProvidesHealing struct {
	Amount Amounter
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

// Stackable はスタック可能なアイテムを示すコンポーネント
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

// EquipmentChanged は装備変更が行われたことを示すダーティーフラグ
type EquipmentChanged struct{}

// Weapon は戦闘中に選択する武器コマンド
type Weapon struct {
	TargetType TargetType
	Cost       int
}

// Attack は攻撃の性質。攻撃毎にこの数値と作用対象のステータスを加味して、最終的なダメージ量を決定する
type Attack struct {
	Accuracy       int         // 命中率
	Damage         int         // 攻撃力
	AttackCount    int         // 攻撃回数
	Element        ElementType // 攻撃属性
	AttackCategory AttackType  // 攻撃種別
}

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

// FactionType は所属派閥。絶対的な指定
type FactionType fmt.Stringer

var (
	// FactionAlly は味方(プレイヤー側)
	FactionAlly FactionType = FactionAllyData{}
	// FactionEnemy は敵性(プレイヤーと敵対)
	FactionEnemy FactionType = FactionEnemyData{}
	// FactionNeutral は中立(会話可能NPC)
	FactionNeutral FactionType = FactionNeutralData{}
)

// FactionAllyData は味方派閥データ
type FactionAllyData struct{}

func (c FactionAllyData) String() string {
	return "FactionAlly"
}

// FactionEnemyData は敵性派閥データ
type FactionEnemyData struct{}

func (c FactionEnemyData) String() string {
	return "FactionEnemy"
}

// FactionNeutralData は中立派閥データ
type FactionNeutralData struct{}

func (c FactionNeutralData) String() string {
	return "FactionNeutral"
}

// ItemLocationType はアイテムの場所
type ItemLocationType fmt.Stringer

var (
	// ItemLocationInBackpack はバックパック内
	ItemLocationInBackpack ItemLocationType = LocationInBackpack{}
	// ItemLocationEquipped は味方が装備中
	ItemLocationEquipped ItemLocationType = LocationEquipped{}
	// ItemLocationOnField はフィールド上
	ItemLocationOnField ItemLocationType = LocationOnField{}
)

// LocationInBackpack はバックパック内位置
type LocationInBackpack struct{}

func (c LocationInBackpack) String() string {
	return "ItemLocationInBackpack"
}

// LocationEquipped は装備中位置
type LocationEquipped struct {
	Owner         ecs.Entity
	EquipmentSlot EquipmentSlotNumber
}

func (c LocationEquipped) String() string {
	return "ItemLocationEquipped"
}

// LocationOnField はフィールド上位置
type LocationOnField struct{}

func (c LocationOnField) String() string {
	return "ItemLocationOnField"
}

// Prop は置物を表すマーカーコンポーネント
type Prop struct{}

// LightSource は光源コンポーネント
type LightSource struct {
	Radius  Tile       // 照明範囲
	Color   color.RGBA // 光の色
	Enabled bool       // 有効/無効
}

// Door は開閉可能なドアコンポーネント
type Door struct {
	IsOpen      bool            // 開いているかどうか
	Orientation DoorOrientation // ドアの向き
}

// DoorOrientation はドアの向き
type DoorOrientation int

const (
	// DoorOrientationHorizontal は横向きのドア
	DoorOrientationHorizontal DoorOrientation = iota
	// DoorOrientationVertical は縦向きのドア
	DoorOrientationVertical
)
