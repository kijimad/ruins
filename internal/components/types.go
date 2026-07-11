package components

import "fmt"

// Pool は最大値と現在値を持つようなパラメータ
type Pool[T int | float64] struct {
	Max     T // 最大値
	Current T // 現在値
}

// IntPool はPool[int]のエイリアス
type IntPool = Pool[int]

// FloatPool はPool[float64]のエイリアス
type FloatPool = Pool[float64]

// TurnBased はエンティティのアクションポイント管理コンポーネント
// ターン制戦闘で、プレイヤー・敵共通で使用される
type TurnBased struct {
	// Action Point
	AP IntPool
	// Speed は毎ターンのAP回復量。能力値・状態異常・装備で変動する
	Speed int
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
	default:
		panic(fmt.Sprintf("未知のEquipmentSlotNumber: %d", s))
	}
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

// HealAmountKind は回復量の指定方法を表す判別子
type HealAmountKind int

const (
	// HealNumeral は絶対量指定
	HealNumeral HealAmountKind = iota
	// HealRatio は最大値に対する倍率指定
	HealRatio
)
