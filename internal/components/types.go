package components

// Pool は最大値と現在値を持つようなパラメータ
type Pool struct {
	Max     int // 計算式で算出される
	Current int // 現在値
}

// PoolFloat はPoolのfloat64精度版
// TODO(kijima): ジェネリクスを検討すべきかもしれない
type PoolFloat struct {
	Max     float64 // 計算式で算出される
	Current float64 // 現在値
}

// TurnBased はエンティティのアクションポイント管理コンポーネント
// ターン制戦闘で、プレイヤー・敵共通で使用される
type TurnBased struct {
	// Action Point
	AP Pool
}

// Attribute は変動するパラメータ値
type Attribute struct {
	Base     int // 固有の値
	Modifier int // 装備などで変動する値
	Total    int // 足し合わせた現在値。算出される値のメモ
}

// TargetType は選択対象
type TargetType struct {
	TargetGroup TargetGroupType // 対象グループ（味方、敵など）
	TargetNum   TargetNumType   // 対象数（単体、複数、全体）
}

// RecipeInput は合成の元になる素材
type RecipeInput struct {
	Name   string // 素材名
	Amount int    // 必要量
}

// EquipBonus は装備品のオプショナルな性能。武器・防具で共通する
type EquipBonus struct {
	Vitality  int // 体力ボーナス
	Strength  int // 筋力ボーナス
	Sensation int // 感覚ボーナス
	Dexterity int // 器用ボーナス
	Agility   int // 敏捷ボーナス

	// 残り項目:
	// - 火属性などの属性耐性
	// - 頑丈+1、連射+2などのスキル
}

// EquipmentSlotNumber は装備スロット番号。0始まり
type EquipmentSlotNumber int

// 装備スロット番号定数
const (
	SlotHead         EquipmentSlotNumber = 0 // 頭部防具スロット
	SlotTorso        EquipmentSlotNumber = 1 // 胴体防具スロット
	SlotLegs         EquipmentSlotNumber = 2 // 脚部防具スロット
	SlotJewelry      EquipmentSlotNumber = 3 // 装飾品スロット
	SlotMeleeWeapon  EquipmentSlotNumber = 4 // 近接武器スロット
	SlotRangedWeapon EquipmentSlotNumber = 5 // 遠距離武器スロット
)

// Amounter は量を計算するためのインターフェース
type Amounter interface {
	Amount() // 量計算を識別するマーカーメソッド
}

var _ Amounter = RatioAmount{}

// RatioAmount は倍率指定
type RatioAmount struct {
	Ratio float64 // 倍率
}

// Amount はAmounterインターフェースの実装
func (ra RatioAmount) Amount() {}

// Calc は倍率と基準値から実際の量を計算する
func (ra RatioAmount) Calc(base int) int {
	return int(float64(base) * ra.Ratio)
}

var _ Amounter = NumeralAmount{}

// NumeralAmount は絶対量指定
type NumeralAmount struct {
	Numeral int // 絶対量
}

// Amount はAmounterインターフェースの実装
func (na NumeralAmount) Amount() {}

// Calc は固定の数値量を返す
func (na NumeralAmount) Calc() int {
	return na.Numeral
}
