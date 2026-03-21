package components

import (
	"errors"
	"fmt"
)

// ErrInvalidEnumType はenumに無効な値が指定された場合のエラー
var ErrInvalidEnumType = errors.New("enumに無効な値が指定された")

// ================

// BodyPart は体の部位を表す
type BodyPart int

// 体の部位定数
const (
	BodyPartHead      BodyPart = iota // 頭
	BodyPartTorso                     // 胴体
	BodyPartArms                      // 腕
	BodyPartHands                     // 手
	BodyPartLegs                      // 脚
	BodyPartFeet                      // 足
	BodyPartWholeBody                 // 全身。体温異常など全身に影響する状態を管理する
	BodyPartCount                     // 部位数
)

// String は部位名を返す
func (bp BodyPart) String() string {
	switch bp {
	case BodyPartHead:
		return "頭"
	case BodyPartTorso:
		return "胴体"
	case BodyPartArms:
		return "腕"
	case BodyPartHands:
		return "手"
	case BodyPartLegs:
		return "脚"
	case BodyPartFeet:
		return "足"
	case BodyPartWholeBody:
		return "全身"
	default:
		panic("不正なBodyPart値")
	}
}

// ================

// TargetNumType はターゲット数を表す
type TargetNumType string

const (
	// TargetSingle は単体ターゲット
	TargetSingle = TargetNumType("SINGLE")
	// TargetAll は全体ターゲット
	TargetAll = TargetNumType("ALL")
)

// Valid はTargetNumTypeの値が有効かを検証する
func (enum TargetNumType) Valid() error {
	switch enum {
	case TargetSingle, TargetAll:
		return nil
	default:
		return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
	}
}

// ================

// TargetGroupType は使用者から見たターゲットの種別。相対的な指定なので、所有者が敵グループだと対象グループは逆転する
type TargetGroupType string

const (
	// TargetGroupAlly は味方グループ
	TargetGroupAlly = TargetGroupType("ALLY") // 味方
	// TargetGroupEnemy は敵グループ
	TargetGroupEnemy = TargetGroupType("ENEMY") // 敵
	// TargetGroupWeapon は武器グループ
	TargetGroupWeapon = TargetGroupType("WEAPON") // 武器
	// TargetGroupNone はグループなし
	TargetGroupNone = TargetGroupType("NONE") // なし
)

// Valid はTargetGroupTypeの値が有効かを検証する
func (enum TargetGroupType) Valid() error {
	switch enum {
	case TargetGroupAlly, TargetGroupEnemy, TargetGroupWeapon, TargetGroupNone:
		return nil
	default:
		return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
	}
}

// ================

// UsableSceneType は使えるシーンを表す
type UsableSceneType string

const (
	// UsableSceneBattle は戦闘シーン
	UsableSceneBattle = UsableSceneType("BATTLE") // 戦闘
	// UsableSceneField はフィールドシーン
	UsableSceneField = UsableSceneType("FIELD") // フィールド
	// UsableSceneAny はいつでも使えるシーン
	UsableSceneAny = UsableSceneType("ANY") // いつでも
)

// Valid はUsableSceneTypeの値が有効かを検証する
func (enum UsableSceneType) Valid() error {
	switch enum {
	case UsableSceneBattle, UsableSceneField, UsableSceneAny:
		return nil
	default:
		return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
	}
}

// ================

// AttackRangeType は攻撃の射程タイプを表す
type AttackRangeType string

const (
	// AttackRangeMelee は近接攻撃
	AttackRangeMelee = AttackRangeType("MELEE")
	// AttackRangeRanged は遠距離攻撃
	AttackRangeRanged = AttackRangeType("RANGED")
)

// AttackType は武器種別を表す。種別によって適用する計算式が異なる
type AttackType struct {
	Type  string          // 武器種別の識別子
	Range AttackRangeType // 近接/遠距離の区分
	Label string          // 表示用ラベル
}

var (
	// AttackSword は刀剣
	AttackSword = AttackType{Type: "SWORD", Range: AttackRangeMelee, Label: "刀剣"}
	// AttackSpear は長物
	AttackSpear = AttackType{Type: "SPEAR", Range: AttackRangeMelee, Label: "長物"}
	// AttackHandgun は拳銃
	AttackHandgun = AttackType{Type: "HANDGUN", Range: AttackRangeRanged, Label: "拳銃"}
	// AttackRifle は小銃
	AttackRifle = AttackType{Type: "RIFLE", Range: AttackRangeRanged, Label: "小銃"}
	// AttackFist は格闘
	AttackFist = AttackType{Type: "FIST", Range: AttackRangeMelee, Label: "格闘"}
	// AttackCanon は大砲
	AttackCanon = AttackType{Type: "CANON", Range: AttackRangeRanged, Label: "大砲"}
)

// AllAttackTypes は定義済みの全AttackTypeのリスト
// 新しいAttackTypeを追加する場合は、ここにも追加すること
var AllAttackTypes = []AttackType{
	AttackSword,
	AttackSpear,
	AttackHandgun,
	AttackRifle,
	AttackFist,
	AttackCanon,
}

// Valid はAttackTypeの値が有効かを検証する
func (at AttackType) Valid() error {
	for _, valid := range AllAttackTypes {
		if at.Type == valid.Type {
			return nil
		}
	}

	return fmt.Errorf("get %s: %w", at.Type, ErrInvalidEnumType)
}

// ParseAttackType は文字列からAttackTypeを生成する
func ParseAttackType(s string) (AttackType, error) {
	for _, at := range AllAttackTypes {
		if at.Type == s {
			return at, nil
		}
	}
	return AttackType{}, fmt.Errorf("invalid attack type: %s: %w", s, ErrInvalidEnumType)
}

// ================

// EquipmentType は装備品種別を表す
// 6部位（頭・胴体・腕・手・脚・足）と装飾品スロット
type EquipmentType string

const (
	// EquipmentHead は頭部装備
	EquipmentHead = EquipmentType("HEAD") // 頭部
	// EquipmentTorso は胴体装備
	EquipmentTorso = EquipmentType("TORSO") // 胴体
	// EquipmentArms は腕装備
	EquipmentArms = EquipmentType("ARMS") // 腕
	// EquipmentHands は手装備
	EquipmentHands = EquipmentType("HANDS") // 手
	// EquipmentLegs は脚装備
	EquipmentLegs = EquipmentType("LEGS") // 脚
	// EquipmentFeet は足装備
	EquipmentFeet = EquipmentType("FEET") // 足
	// EquipmentJewelry はアクセサリ装備
	EquipmentJewelry = EquipmentType("JEWELRY") // アクセサリ
)

// Valid はEquipmentTypeの値が有効かを検証する
func (enum EquipmentType) Valid() error {
	switch enum {
	case EquipmentHead, EquipmentTorso, EquipmentArms, EquipmentHands,
		EquipmentLegs, EquipmentFeet, EquipmentJewelry:
		return nil
	default:
		return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
	}
}

// SlotNumber はEquipmentTypeに対応するEquipmentSlotNumberを返す
func (enum EquipmentType) SlotNumber() EquipmentSlotNumber {
	switch enum {
	case EquipmentHead:
		return SlotHead
	case EquipmentTorso:
		return SlotTorso
	case EquipmentArms:
		return SlotArms
	case EquipmentHands:
		return SlotHands
	case EquipmentLegs:
		return SlotLegs
	case EquipmentFeet:
		return SlotFeet
	case EquipmentJewelry:
		return SlotJewelry
	}
	panic(fmt.Sprintf("不正なEquipmentType値: %s", string(enum)))
}

func (enum EquipmentType) String() string {
	switch enum {
	case EquipmentHead:
		return "頭部"
	case EquipmentTorso:
		return "胴体"
	case EquipmentArms:
		return "腕部"
	case EquipmentHands:
		return "手部"
	case EquipmentLegs:
		return "脚部"
	case EquipmentFeet:
		return "足部"
	case EquipmentJewelry:
		return "装飾"
	}
	panic(fmt.Sprintf("不正なEquipmentType値: %s", string(enum)))
}

// ================

// ElementType は攻撃属性を表す
type ElementType string

const (
	// ElementTypeNone は属性なし
	ElementTypeNone ElementType = "NONE"
	// ElementTypeFire は火属性
	ElementTypeFire ElementType = "FIRE"
	// ElementTypeThunder は雷属性
	ElementTypeThunder ElementType = "THUNDER"
	// ElementTypeChill は氷属性
	ElementTypeChill ElementType = "CHILL"
	// ElementTypePhoton は光属性
	ElementTypePhoton ElementType = "PHOTON"
)

// Valid はElementTypeの値が有効かを検証する
func (enum ElementType) Valid() error {
	switch enum {
	case ElementTypeNone, ElementTypeFire, ElementTypeThunder, ElementTypeChill, ElementTypePhoton:
		return nil
	default:
		return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
	}
}

func (enum ElementType) String() string {
	switch enum {
	case ElementTypeNone:
		return "無"
	case ElementTypeFire:
		return "火"
	case ElementTypeThunder:
		return "電"
	case ElementTypeChill:
		return "冷"
	case ElementTypePhoton:
		return "光"
	}
	panic(fmt.Sprintf("不正なElementType値: %s", string(enum)))
}
