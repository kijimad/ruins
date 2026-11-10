package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// Pool は最大値と現在値を持つようなパラメータ
type Pool[T consts.Numeric] struct {
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
	AP IntPool
	// Speed は毎ターンのAP回復量。能力値・状態異常・装備で変動する
	Speed int
}

// TargetType は選択対象
type TargetType struct {
	TargetGroup oapi.TargetGroup // 対象グループ（味方、敵など）
	TargetNum   oapi.TargetNum   // 対象数（単体、複数、全体）
}

// RecipeInput はクラフトの元になる素材
type RecipeInput struct {
	ID     string // 素材アイテムの id
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
		return EquipmentCategoryName(oapi.EquipmentCategoryHEAD)
	case SlotTorso:
		return EquipmentCategoryName(oapi.EquipmentCategoryTORSO)
	case SlotArms:
		return EquipmentCategoryName(oapi.EquipmentCategoryARMS)
	case SlotHands:
		return EquipmentCategoryName(oapi.EquipmentCategoryHANDS)
	case SlotLegs:
		return EquipmentCategoryName(oapi.EquipmentCategoryLEGS)
	case SlotFeet:
		return EquipmentCategoryName(oapi.EquipmentCategoryFEET)
	case SlotJewelry:
		return EquipmentCategoryName(oapi.EquipmentCategoryJEWELRY)
	case SlotWeapon1:
		return "Weapon 1"
	case SlotWeapon2:
		return "Weapon 2"
	case SlotWeapon3:
		return "Weapon 3"
	case SlotWeapon4:
		return "Weapon 4"
	case SlotWeapon5:
		return "Weapon 5"
	default:
		panic(fmt.Sprintf("unknown EquipmentSlotNumber: %d", s))
	}
}

// ParseEquipmentSlot は raw の装備スロットから装備スロット番号を返す。未知の値は ok=false
func ParseEquipmentSlot(slot oapi.EquipSlot) (EquipmentSlotNumber, bool) {
	switch slot {
	case oapi.EquipSlotWEAPON1:
		return SlotWeapon1, true
	case oapi.EquipSlotWEAPON2:
		return SlotWeapon2, true
	case oapi.EquipSlotWEAPON3:
		return SlotWeapon3, true
	case oapi.EquipSlotWEAPON4:
		return SlotWeapon4, true
	case oapi.EquipSlotWEAPON5:
		return SlotWeapon5, true
	case oapi.EquipSlotHEAD:
		return SlotHead, true
	case oapi.EquipSlotTORSO:
		return SlotTorso, true
	case oapi.EquipSlotARMS:
		return SlotArms, true
	case oapi.EquipSlotHANDS:
		return SlotHands, true
	case oapi.EquipSlotLEGS:
		return SlotLegs, true
	case oapi.EquipSlotFEET:
		return SlotFeet, true
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
