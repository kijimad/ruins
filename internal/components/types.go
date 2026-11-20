package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
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
	TargetGroup rawschema.TargetGroup // 対象グループ（味方、敵など）
	TargetNum   rawschema.TargetNum   // 対象数（単体、複数、全体）
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
	SlotHand1   EquipmentSlotNumber = 7  // 手持ち欄1
	SlotHand2   EquipmentSlotNumber = 8  // 手持ち欄2
	SlotHand3   EquipmentSlotNumber = 9  // 手持ち欄3
	SlotHand4   EquipmentSlotNumber = 10 // 手持ち欄4
	SlotHand5   EquipmentSlotNumber = 11 // 手持ち欄5
)

// HandSlotCount は手持ち欄の数
const HandSlotCount = int(SlotHand5-SlotHand1) + 1

// HandIndex は手持ち欄の、SlotHand1 から並べた並びでの添字を返す。手持ち欄でなければ ok は偽
func (s EquipmentSlotNumber) HandIndex() (int, bool) {
	if s < SlotHand1 || s > SlotHand5 {
		return 0, false
	}
	return int(s - SlotHand1), true
}

// Name は装備スロット番号の表示名を返す
func (s EquipmentSlotNumber) Name() i18n.Msgid {
	switch s {
	case SlotHead:
		return EquipmentCategoryName(rawschema.EquipmentCategoryHEAD)
	case SlotTorso:
		return EquipmentCategoryName(rawschema.EquipmentCategoryTORSO)
	case SlotArms:
		return EquipmentCategoryName(rawschema.EquipmentCategoryARMS)
	case SlotHands:
		return EquipmentCategoryName(rawschema.EquipmentCategoryHANDS)
	case SlotLegs:
		return EquipmentCategoryName(rawschema.EquipmentCategoryLEGS)
	case SlotFeet:
		return EquipmentCategoryName(rawschema.EquipmentCategoryFEET)
	case SlotJewelry:
		return EquipmentCategoryName(rawschema.EquipmentCategoryJEWELRY)
	case SlotHand1:
		return "Hand 1"
	case SlotHand2:
		return "Hand 2"
	case SlotHand3:
		return "Hand 3"
	case SlotHand4:
		return "Hand 4"
	case SlotHand5:
		return "Hand 5"
	default:
		panic(fmt.Sprintf("unknown EquipmentSlotNumber: %d", s))
	}
}

// ParseEquipmentSlot は raw の装備スロットから装備スロット番号を返す。未知の値は ok=false
func ParseEquipmentSlot(slot rawschema.EquipSlot) (EquipmentSlotNumber, bool) {
	switch slot {
	case rawschema.EquipSlotHAND1:
		return SlotHand1, true
	case rawschema.EquipSlotHAND2:
		return SlotHand2, true
	case rawschema.EquipSlotHAND3:
		return SlotHand3, true
	case rawschema.EquipSlotHAND4:
		return SlotHand4, true
	case rawschema.EquipSlotHAND5:
		return SlotHand5, true
	case rawschema.EquipSlotHEAD:
		return SlotHead, true
	case rawschema.EquipSlotTORSO:
		return SlotTorso, true
	case rawschema.EquipSlotARMS:
		return SlotArms, true
	case rawschema.EquipSlotHANDS:
		return SlotHands, true
	case rawschema.EquipSlotLEGS:
		return SlotLegs, true
	case rawschema.EquipSlotFEET:
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
