package components

import (
	"errors"
	"fmt"

	"github.com/kijimaD/ruins/internal/oapi"
)

// ErrInvalidEnumType はinvalid enum value場合のエラー
var ErrInvalidEnumType = errors.New("invalid enum value")

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

// String は raw の bodyParts の id を返す。表示名は query.BodyPartName で引く
func (bp BodyPart) String() string {
	return bodyPartMetas[bp].id
}

// ID は raw の bodyParts の id を返す
func (bp BodyPart) ID() string {
	return bodyPartMetas[bp].id
}

// BodyPartByID は raw の bodyParts の id から部位を返す。該当が無ければ ok=false
func BodyPartByID(id string) (BodyPart, bool) {
	for bp := range BodyPartCount {
		if bodyPartMetas[bp].id == id {
			return bp, true
		}
	}
	return 0, false
}

// ================

// ================

// ================

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
	AttackSword = AttackType{Type: "sword", Range: AttackRangeMelee, Label: "Sword"}
	// AttackSpear は長物
	AttackSpear = AttackType{Type: "spear", Range: AttackRangeMelee, Label: "Spear"}
	// AttackHandgun は拳銃
	AttackHandgun = AttackType{Type: "handgun", Range: AttackRangeRanged, Label: "Handgun"}
	// AttackRifle は小銃
	AttackRifle = AttackType{Type: "rifle", Range: AttackRangeRanged, Label: "Rifle"}
	// AttackFist は格闘
	AttackFist = AttackType{Type: "fist", Range: AttackRangeMelee, Label: "Martial arts"}
	// AttackCannon は大砲
	AttackCannon = AttackType{Type: "cannon", Range: AttackRangeRanged, Label: "Cannon"}
	// AttackBow は弓
	AttackBow = AttackType{Type: "bow", Range: AttackRangeRanged, Label: "Bow"}
)

// AllAttackTypes は定義済みの全AttackTypeのリスト
// 新しいAttackTypeを追加する場合は、ここにも追加すること
var AllAttackTypes = []AttackType{
	AttackSword,
	AttackSpear,
	AttackHandgun,
	AttackRifle,
	AttackFist,
	AttackCannon,
	AttackBow,
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

// RangeParams は遠距離武器の射程パラメータ
type RangeParams struct {
	OptimalRange   int // 最適射程。この距離内はペナルティなし
	MaxRange       int // 最大射程。超過で射撃不可
	PenaltyPerTile int // 最適射程超過時の1タイルあたり命中率ペナルティ(%)
}

// rangeParamsMap は武器種別ごとの射程パラメータ
var rangeParamsMap = map[string]RangeParams{
	AttackBow.Type:     {OptimalRange: 4, MaxRange: 10, PenaltyPerTile: 5},
	AttackHandgun.Type: {OptimalRange: 3, MaxRange: 8, PenaltyPerTile: 8},
	AttackRifle.Type:   {OptimalRange: 8, MaxRange: 16, PenaltyPerTile: 3},
	AttackCannon.Type:  {OptimalRange: 6, MaxRange: 12, PenaltyPerTile: 5},
}

// GetRangeParams は武器種別の射程パラメータを返す。未定義ならfalseを返す
func GetRangeParams(attackType AttackType) (RangeParams, bool) {
	params, ok := rangeParamsMap[attackType.Type]
	return params, ok
}

// ================

// EquipmentCategoryName は防具の部位の表示名 msgid を返す
func EquipmentCategoryName(c oapi.EquipmentCategory) string {
	switch c {
	case oapi.EquipmentCategoryHEAD:
		return "Head"
	case oapi.EquipmentCategoryTORSO:
		return "Torso"
	case oapi.EquipmentCategoryARMS:
		return "Arms"
	case oapi.EquipmentCategoryHANDS:
		return "Hands"
	case oapi.EquipmentCategoryLEGS:
		return "Legs"
	case oapi.EquipmentCategoryFEET:
		return "Feet"
	case oapi.EquipmentCategoryJEWELRY:
		return "Accessory"
	}
	panic(fmt.Sprintf("invalid EquipmentCategory value: %s", string(c)))
}

// ================

// ElementName は攻撃属性の表示名 msgid を返す
func ElementName(e oapi.Element) string {
	switch e {
	case oapi.ElementNONE:
		return "None"
	case oapi.ElementFIRE:
		return "Fire"
	case oapi.ElementTHUNDER:
		return "Thunder"
	case oapi.ElementCHILL:
		return "Ice"
	case oapi.ElementPHOTON:
		return "Light"
	}
	panic(fmt.Sprintf("invalid Element value: %s", string(e)))
}
