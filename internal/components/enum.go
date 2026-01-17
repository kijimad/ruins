package components

import (
	"errors"
	"fmt"
)

// ErrInvalidEnumType はenumに無効な値が指定された場合のエラー
var ErrInvalidEnumType = errors.New("enumに無効な値が指定された")

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
	}

	return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
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
	}

	return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
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
	}

	return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
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
type EquipmentType string

const (
	// EquipmentHead は頭部装備
	EquipmentHead = EquipmentType("HEAD") // 頭部
	// EquipmentTorso は胴体装備
	EquipmentTorso = EquipmentType("TORSO") // 胴体
	// EquipmentLegs は脚装備
	EquipmentLegs = EquipmentType("LEGS") // 脚
	// EquipmentJewelry はアクセサリ装備
	EquipmentJewelry = EquipmentType("JEWELRY") // アクセサリ
)

// Valid はEquipmentTypeの値が有効かを検証する
func (enum EquipmentType) Valid() error {
	switch enum {
	case EquipmentHead, EquipmentTorso, EquipmentLegs, EquipmentJewelry:
		return nil
	}

	return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
}

func (enum EquipmentType) String() string {
	var result string
	switch enum {
	case EquipmentHead:
		result = "頭部"
	case EquipmentTorso:
		result = "胴体"
	case EquipmentLegs:
		result = "脚部"
	case EquipmentJewelry:
		result = "装飾"
	default:
		panic("invalid equiment slot type")
	}
	return result
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
	}
	return fmt.Errorf("get %s: %w", enum, ErrInvalidEnumType)
}

func (enum ElementType) String() string {
	var result string
	switch enum {
	case ElementTypeNone:
		result = "無"
	case ElementTypeFire:
		result = "火"
	case ElementTypeThunder:
		result = "電"
	case ElementTypeChill:
		result = "冷"
	case ElementTypePhoton:
		result = "光"
	default:
		panic("invalid element type")
	}
	return result
}
