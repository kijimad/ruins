package components

import (
	"fmt"
	"image/color"
	"reflect"

	"github.com/kijimaD/ruins/internal/consts"
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
	HP             *HP
	Consumable     *Consumable
	WeightCapacity *WeightCapacity
	Melee          *Melee
	Fire           *Fire
	Value          *Value
	Weight         *Weight
	Recipe         *Recipe
	Wearable       *Wearable
	Abilities      *Abilities
	Ammo           *Ammo
	Stackable      *Stackable
	Material       *Material
	LocationType   *LocationType

	// field ================
	Tile            *Tile
	AI              *AI
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
	FactionType   *FactionType
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

// Component は *ecs.SliceComponent をラップし、型付きの取得を提供する。
// *ecs.SliceComponent を埋め込むため、Join/AddComponent 等が要求する
// ecs.DataComponent/joinable インターフェースは昇格したメソッドで満たす。
// 埋め込みの Get(ecs.Entity) interface{} は残す（DataComponent が要求するため）。
type Component[T any] struct {
	*ecs.SliceComponent
}

// MustGet はエンティティの当該コンポーネントを型付きで取得する。
// 保持していない場合はpanicする（呼び出し側が保持を保証している場合に使う）。
func (c Component[T]) MustGet(entity ecs.Entity) *T {
	v, ok := c.Get(entity).(*T)
	if !ok {
		panic(fmt.Sprintf("MustGet: エンティティ %v が %s を保持していない", entity, reflect.TypeFor[T]()))
	}
	return v
}

// TryGet はエンティティの当該コンポーネントを型付きで取得する。
// 保持していない場合は (nil, false) を返す。
// 保持しているが型が異なる場合はプログラミングエラーのため panic する。
func (c Component[T]) TryGet(entity ecs.Entity) (*T, bool) {
	v := c.Get(entity)
	if v == nil {
		return nil, false
	}
	comp, ok := v.(*T)
	if !ok {
		panic(fmt.Sprintf("TryGet: エンティティ %v が保持する値が %s ではない", entity, reflect.TypeFor[T]()))
	}
	return comp, true
}

// AddComponent はエンティティに型付きでコンポーネントを付与する。
// data の型が *T に縛られるため、フィールドとデータ型の取り違えをコンパイラが検出する。
func AddComponent[T any](entity ecs.Entity, comp Component[T], data *T) {
	entity.AddComponent(comp, data)
}

// initSlice は内部の SliceComponent を初期化する。InitializeComponents から reflect 経由で呼ばれる
func (c *Component[T]) initSlice(manager *ecs.Manager) {
	c.SliceComponent = manager.NewSliceComponent()
}

// sliceComponentIniter は Component[T] を型に依らず初期化するためのマーカーインターフェース
type sliceComponentIniter interface {
	initSlice(*ecs.Manager)
}

// Components はECSコンポーネントストレージ
// 各コンポーネント型のComponent[T]/NullComponentを保持し、
// Manager.Join()でのクエリに使用される。
// 各コンポーネントの型付き取得は Component[T] の MustGet/TryGet を使う
type Components struct {
	// general ================
	Name        Component[Name]
	Description Component[Description]

	// item ================
	HP                 Component[HP]
	Consumable         Component[Consumable]
	WeightCapacity     Component[WeightCapacity]
	Melee              Component[Melee]
	Fire               Component[Fire]
	Value              Component[Value]
	Weight             Component[Weight]
	Recipe             Component[Recipe]
	Wearable           Component[Wearable]
	Abilities          Component[Abilities]
	Ammo               Component[Ammo]
	Stackable          Component[Stackable]
	Material           *ecs.NullComponent
	LocationInBackpack Component[LocationInBackpack]
	LocationEquipped   Component[LocationEquipped]
	LocationOnField    *ecs.NullComponent
	LocationInStorage  Component[LocationInStorage]

	// field ================
	Tile            *ecs.NullComponent
	AI              Component[AI]
	Camera          Component[Camera]
	Position        Component[Position]
	GridElement     Component[GridElement]
	SpriteRender    Component[SpriteRender]
	BlockView       *ecs.NullComponent
	BlockPass       *ecs.NullComponent
	PassCost        Component[PassCost]
	Door            Component[Door]
	Prop            *ecs.NullComponent
	LightSource     Component[LightSource]
	Interactable    Component[Interactable]
	VisualEffect    Component[VisualEffects]
	TileTemperature Component[TileTemperature]

	// member ================
	Player         *ecs.NullComponent
	Profession     Component[Profession]
	Hunger         Component[Hunger]
	Wallet         Component[Wallet]
	FactionAlly    *ecs.NullComponent
	FactionEnemy   *ecs.NullComponent
	FactionNeutral *ecs.NullComponent
	Boss           *ecs.NullComponent // ボスエンティティのマーカー
	Dialog         Component[Dialog]
	Dead           *ecs.NullComponent
	TurnBased      Component[TurnBased]
	HealthStatus   Component[HealthStatus]
	Skills         Component[Skills]
	CharModifiers  Component[CharModifiers]

	// event ================
	StateChangeRequest Component[StateChangeRequest] // ステート遷移リクエスト
	StatsChanged       *ecs.NullComponent
	WeightDirty        *ecs.NullComponent
	ProvidesHealing    Component[ProvidesHealing]
	ProvidesNutrition  Component[ProvidesNutrition]
	InflictsDamage     Component[InflictsDamage]

	// book ================
	Book Component[Book]

	// battle ================
	CommandTable Component[CommandTable]
	DropTable    Component[DropTable]

	// squad ================
	SquadMember Component[SquadMember]

	// activity ================
	Activity     Component[Activity]     // 実行中のアクティビティ
	LastActivity Component[LastActivity] // 直近のアクティビティ実行結果

	// singleton ================
	GameLog      Component[GameLog]      // フィールドログストレージ
	DungeonState Component[Dungeon]      // ダンジョン状態
	GameProgress Component[GameProgress] // ゲーム進行データ
	TurnState    Component[TurnState]    // ターン状態
	SpatialIndex Component[SpatialIndex] // 空間インデックス
}

// InitializeComponents はComponentInitializerインターフェースを実装する
// リフレクションを使用して自動的に全コンポーネントを初期化する
// コンポーネント追加時の手動更新が不要
func (c *Components) InitializeComponents(manager *ecs.Manager) error {
	val := reflect.ValueOf(c).Elem() // *Components から Components へ
	typ := val.Type()

	for i := range val.NumField() {
		if err := initComponentField(val.Field(i), typ.Field(i).Name, manager); err != nil {
			return err
		}
	}

	return nil
}

// initComponentField は1つのコンポーネントフィールドを型に応じて初期化する。
// エラーパスを単体テストできるよう、フィールド単位の処理を切り出している。
func initComponentField(field reflect.Value, fieldName string, manager *ecs.Manager) error {
	// フィールドが設定可能かチェック
	if !field.CanSet() {
		return fmt.Errorf("field %s is not settable", fieldName)
	}

	// フィールドの種類に応じて初期化する。
	// Component[T] はジェネリックで各インスタンスが別型のため、型switchではなく
	// マーカーインターフェース経由で初期化する。
	switch f := field.Addr().Interface().(type) {
	case sliceComponentIniter:
		// Component[T] の内部 SliceComponent を初期化
		f.initSlice(manager)
	case **ecs.NullComponent:
		// NullComponent の初期化
		*f = manager.NewNullComponent()
	default:
		// 未対応の型はエラーとして扱う
		return fmt.Errorf("unsupported component type %v for field %s", field.Type(), fieldName)
	}

	return nil
}

// Camera はカメラ
// 滑らかなズームと追従のため、実際値と目標値を別々に持つ
type Camera struct {
	// ズーム率
	Scale   float64
	ScaleTo float64
	// カメラ位置。ピクセル単位
	X       float64
	Y       float64
	TargetX float64
	TargetY float64
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

// LocationType はエンティティの場所
type LocationType fmt.Stringer

var (
	// LocationTypeInBackpack はバックパック内
	LocationTypeInBackpack LocationType = LocationInBackpack{}
	// LocationTypeEquipped は装備中
	LocationTypeEquipped LocationType = LocationEquipped{}
	// LocationTypeOnField はフィールド上
	LocationTypeOnField LocationType = LocationOnField{}
	// LocationTypeInStorage は収納内
	LocationTypeInStorage LocationType = LocationInStorage{}
)

// Location はエンティティの位置を表すインターフェース。
// setLocationの引数を型安全にするためのマーカー
type Location interface {
	isLocation()
}

// LocationInBackpack はバックパック内位置
type LocationInBackpack struct {
	Owner ecs.Entity // バックパックの所有者
}

func (c LocationInBackpack) String() string { return "LocationInBackpack" }
func (c LocationInBackpack) isLocation()    {}

// LocationEquipped は装備中位置
type LocationEquipped struct {
	Owner         ecs.Entity
	EquipmentSlot EquipmentSlotNumber
}

func (c LocationEquipped) String() string { return "LocationEquipped" }
func (c LocationEquipped) isLocation()    {}

// LocationOnField はフィールド上位置
type LocationOnField struct{}

func (c LocationOnField) String() string { return "LocationOnField" }
func (c LocationOnField) isLocation()    {}

// LocationInStorage は収納内位置
type LocationInStorage struct {
	Owner ecs.Entity // 収納Propのエンティティ
}

func (c LocationInStorage) String() string { return "LocationInStorage" }
func (c LocationInStorage) isLocation()    {}

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
