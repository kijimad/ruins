package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyPart_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		bp   BodyPart
		want string
	}{
		{BodyPartHead, "頭"},
		{BodyPartTorso, "胴体"},
		{BodyPartArms, "腕"},
		{BodyPartHands, "手"},
		{BodyPartLegs, "脚"},
		{BodyPartFeet, "足"},
		{BodyPartWholeBody, "全身"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.bp.String())
		})
	}
}

func TestBodyPart_String_InvalidPanic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = BodyPart(99).String()
	})
}

func TestParseAttackType(t *testing.T) {
	t.Parallel()

	t.Run("有効な攻撃タイプ", func(t *testing.T) {
		t.Parallel()
		for _, at := range AllAttackTypes {
			result, err := ParseAttackType(at.Type)
			require.NoError(t, err)
			assert.Equal(t, at, result)
		}
	})

	t.Run("無効な攻撃タイプ", func(t *testing.T) {
		t.Parallel()
		_, err := ParseAttackType("INVALID")
		assert.ErrorIs(t, err, ErrInvalidEnumType)
	})
}

func TestGetRangeParams(t *testing.T) {
	t.Parallel()

	t.Run("遠距離武器には射程パラメータがある", func(t *testing.T) {
		t.Parallel()
		for _, at := range []AttackType{AttackBow, AttackHandgun, AttackRifle, AttackCanon} {
			params, ok := GetRangeParams(at)
			assert.True(t, ok, at.Label)
			assert.Greater(t, params.MaxRange, 0, at.Label)
			assert.Greater(t, params.OptimalRange, 0, at.Label)
		}
	})

	t.Run("近接武器には射程パラメータがない", func(t *testing.T) {
		t.Parallel()
		_, ok := GetRangeParams(AttackSword)
		assert.False(t, ok)
	})
}

func TestEquipmentType_Valid(t *testing.T) {
	t.Parallel()

	validTypes := []EquipmentType{
		EquipmentHead, EquipmentTorso, EquipmentArms,
		EquipmentHands, EquipmentLegs, EquipmentFeet, EquipmentJewelry,
	}
	for _, et := range validTypes {
		assert.NoError(t, et.Valid(), string(et))
	}

	assert.ErrorIs(t, EquipmentType("INVALID").Valid(), ErrInvalidEnumType)
}

func TestEquipmentType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		et   EquipmentType
		want string
	}{
		{EquipmentHead, "頭部"},
		{EquipmentTorso, "胴体"},
		{EquipmentArms, "腕部"},
		{EquipmentHands, "手部"},
		{EquipmentLegs, "脚部"},
		{EquipmentFeet, "足部"},
		{EquipmentJewelry, "装飾"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.et.String())
		})
	}
}

func TestEquipmentType_SlotNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		et   EquipmentType
		want EquipmentSlotNumber
	}{
		{EquipmentHead, SlotHead},
		{EquipmentTorso, SlotTorso},
		{EquipmentArms, SlotArms},
		{EquipmentHands, SlotHands},
		{EquipmentLegs, SlotLegs},
		{EquipmentFeet, SlotFeet},
		{EquipmentJewelry, SlotJewelry},
	}

	for _, tt := range tests {
		t.Run(string(tt.et), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.et.SlotNumber())
		})
	}
}

func TestElementType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		elem ElementType
		want string
	}{
		{ElementTypeNone, "無"},
		{ElementTypeFire, "火"},
		{ElementTypeThunder, "雷"},
		{ElementTypeChill, "氷"},
		{ElementTypePhoton, "光"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.elem.String())
		})
	}
}

func TestElementType_String_InvalidPanic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = ElementType("INVALID").String()
	})
}
