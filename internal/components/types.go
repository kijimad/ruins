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
	// Speed は毎ターンのAP回復量。能力値・状態異常・装備で変動する
	Speed int
}

// Ability は変動するパラメータ値
type Ability struct {
	Base     int // 固有の値
	Modifier int // 装備や健康状態で変動する値
	Total    int // 計算した現在値。算出される値のメモ
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
	SlotHead    EquipmentSlotNumber = 0  // 頭部防具スロット
	SlotTorso   EquipmentSlotNumber = 1  // 胴体防具スロット
	SlotArms    EquipmentSlotNumber = 2  // 腕部防具スロット
	SlotHands   EquipmentSlotNumber = 3  // 手部防具スロット
	SlotLegs    EquipmentSlotNumber = 4  // 脚部防具スロット
	SlotFeet    EquipmentSlotNumber = 5  // 足部防具スロット
	SlotJewelry EquipmentSlotNumber = 6  // 装飾品スロット
	SlotWeapon1 EquipmentSlotNumber = 7  // 武器スロット1
	SlotWeapon2 EquipmentSlotNumber = 8  // 武器スロット2
	SlotWeapon3 EquipmentSlotNumber = 9  // 武器スロット3
	SlotWeapon4 EquipmentSlotNumber = 10 // 武器スロット4
	SlotWeapon5 EquipmentSlotNumber = 11 // 武器スロット5
)

// String は装備スロット番号の表示名を返す
func (s EquipmentSlotNumber) String() string {
	switch s {
	case SlotHead:
		return EquipmentHead.String()
	case SlotTorso:
		return EquipmentTorso.String()
	case SlotArms:
		return EquipmentArms.String()
	case SlotHands:
		return EquipmentHands.String()
	case SlotLegs:
		return EquipmentLegs.String()
	case SlotFeet:
		return EquipmentFeet.String()
	case SlotJewelry:
		return EquipmentJewelry.String()
	case SlotWeapon1:
		return "武器1"
	case SlotWeapon2:
		return "武器2"
	case SlotWeapon3:
		return "武器3"
	case SlotWeapon4:
		return "武器4"
	case SlotWeapon5:
		return "武器5"
	}
	return "不明"
}

// ParseEquipmentSlot は文字列から装備スロット番号を返す。
// 防具スロットはEquipmentTypeの定数値、武器スロットは"WEAPON1"〜"WEAPON5"を受け付ける。
func ParseEquipmentSlot(s string) (EquipmentSlotNumber, bool) {
	// 武器スロット
	switch s {
	case "WEAPON1":
		return SlotWeapon1, true
	case "WEAPON2":
		return SlotWeapon2, true
	case "WEAPON3":
		return SlotWeapon3, true
	case "WEAPON4":
		return SlotWeapon4, true
	case "WEAPON5":
		return SlotWeapon5, true
	}

	// 防具スロット: EquipmentTypeを経由して変換する
	et := EquipmentType(s)
	if et.Valid() == nil {
		return et.SlotNumber(), true
	}
	return 0, false
}

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
