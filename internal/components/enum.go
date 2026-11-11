package components

import (
	"errors"
	"fmt"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/rawschema"
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

// RangeParams は遠距離武器の射程パラメータ。raw の attackTypes の行から組み立てる
type RangeParams struct {
	OptimalRange   int // 最適射程。この距離内はペナルティなし
	MaxRange       int // 最大射程。超過で射撃不可
	PenaltyPerTile int // 最適射程超過時の1タイルあたり命中率ペナルティ(%)
}

// ================

// EquipmentCategoryName は防具の部位の表示名 msgid を返す
func EquipmentCategoryName(c rawschema.EquipmentCategory) i18n.Msgid {
	switch c {
	case rawschema.EquipmentCategoryHEAD:
		return "Head"
	case rawschema.EquipmentCategoryTORSO:
		return "Torso"
	case rawschema.EquipmentCategoryARMS:
		return "Arms"
	case rawschema.EquipmentCategoryHANDS:
		return "Hands"
	case rawschema.EquipmentCategoryLEGS:
		return "Legs"
	case rawschema.EquipmentCategoryFEET:
		return "Feet"
	case rawschema.EquipmentCategoryJEWELRY:
		return "Accessory"
	}
	panic(fmt.Sprintf("invalid EquipmentCategory value: %s", string(c)))
}

// ================

// ElementName は攻撃属性の表示名 msgid を返す
func ElementName(e rawschema.Element) i18n.Msgid {
	switch e {
	case rawschema.ElementNONE:
		return "None"
	case rawschema.ElementFIRE:
		return "Fire"
	case rawschema.ElementTHUNDER:
		return "Thunder"
	case rawschema.ElementCHILL:
		return "Ice"
	case rawschema.ElementPHOTON:
		return "Light"
	}
	panic(fmt.Sprintf("invalid Element value: %s", string(e)))
}
