package components

import (
	"errors"
	"fmt"
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

// String は部位名を返す。対応は bodyPartMetas で定める
func (bp BodyPart) String() string {
	return bodyPartMetas[bp].displayName
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
	AttackSword = AttackType{Type: "SWORD", Range: AttackRangeMelee, Label: "Sword"}
	// AttackSpear は長物
	AttackSpear = AttackType{Type: "SPEAR", Range: AttackRangeMelee, Label: "Spear"}
	// AttackHandgun は拳銃
	AttackHandgun = AttackType{Type: "HANDGUN", Range: AttackRangeRanged, Label: "Handgun"}
	// AttackRifle は小銃
	AttackRifle = AttackType{Type: "RIFLE", Range: AttackRangeRanged, Label: "Rifle"}
	// AttackFist は格闘
	AttackFist = AttackType{Type: "FIST", Range: AttackRangeMelee, Label: "Martial arts"}
	// AttackCanon は大砲
	AttackCanon = AttackType{Type: "CANON", Range: AttackRangeRanged, Label: "Cannon"}
	// AttackBow は弓
	AttackBow = AttackType{Type: "BOW", Range: AttackRangeRanged, Label: "Bow"}
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
	AttackCanon.Type:   {OptimalRange: 6, MaxRange: 12, PenaltyPerTile: 5},
}

// GetRangeParams は武器種別の射程パラメータを返す。未定義ならfalseを返す
func GetRangeParams(attackType AttackType) (RangeParams, bool) {
	params, ok := rangeParamsMap[attackType.Type]
	return params, ok
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
	panic(fmt.Sprintf("invalid EquipmentType value: %s", string(enum)))
}

func (enum EquipmentType) String() string {
	switch enum {
	case EquipmentHead:
		return "Head"
	case EquipmentTorso:
		return "Torso"
	case EquipmentArms:
		return "Arms"
	case EquipmentHands:
		return "Hands"
	case EquipmentLegs:
		return "Legs"
	case EquipmentFeet:
		return "Feet"
	case EquipmentJewelry:
		return "Accessory"
	}
	panic(fmt.Sprintf("invalid EquipmentType value: %s", string(enum)))
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

func (enum ElementType) String() string {
	switch enum {
	case ElementTypeNone:
		return "None"
	case ElementTypeFire:
		return "Fire"
	case ElementTypeThunder:
		return "Thunder"
	case ElementTypeChill:
		return "Ice"
	case ElementTypePhoton:
		return "Light"
	}
	panic(fmt.Sprintf("invalid ElementType value: %s", string(enum)))
}
