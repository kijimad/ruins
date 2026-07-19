package components

import (
	"fmt"
	"image/color"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/mlange-42/ark/ecs"
)

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

// Camera はカメラ
// 滑らかなズームと追従のため、実際値と目標値を別々に持つ
type Camera struct {
	// ズーム率
	Scale   float64
	ScaleTo float64
	// カメラ位置。ワールド空間のピクセル単位
	Pos    consts.Coord[consts.Pixel]
	Target consts.Coord[consts.Pixel]
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

// 派閥は種別ごとに独立したマーカーコンポーネントとして表現する。
// archetypeクエリ（「全敵」「全味方」）が効くため archetype ECS に適している。
// 派閥は排他（1エンティティは高々1つ）で、生成時に EntitySpec で1つだけ指定する。

// 派閥のenum文字列（oapiのFactionType enumと一致させる）
const (
	FactionAllyName    = "FactionAlly"
	FactionEnemyName   = "FactionEnemy"
	FactionNeutralName = "FactionNeutral"
)

// FactionAlly は味方派閥(プレイヤー側)のマーカー
type FactionAlly struct{}

// FactionEnemy は敵性派閥(プレイヤーと敵対)のマーカー
type FactionEnemy struct{}

// FactionNeutral は中立派閥(会話可能NPC)のマーカー
type FactionNeutral struct{}

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
