package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBodyPart_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		bp   BodyPart
		want string
	}{
		{BodyPartHead, "head"},
		{BodyPartTorso, "torso"},
		{BodyPartArms, "arm"},
		{BodyPartHands, "hand"},
		{BodyPartLegs, "leg"},
		{BodyPartFeet, "foot"},
		{BodyPartWholeBody, "whole_body"},
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
		for _, at := range []AttackType{AttackBow, AttackHandgun, AttackRifle, AttackCannon} {
			params, ok := GetRangeParams(at)
			assert.True(t, ok, at.Label)
			assert.Positive(t, params.MaxRange, at.Label)
			assert.Positive(t, params.OptimalRange, at.Label)
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
		require.NoError(t, et.Valid(), string(et))
	}

	assert.ErrorIs(t, EquipmentType("INVALID").Valid(), ErrInvalidEnumType)
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

func TestEquipmentType_SlotNumber_不正な値はpanicする(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = EquipmentType("INVALID").SlotNumber()
	})
}

func TestElementName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		et   oapi.Element
		want string
	}{
		{oapi.ElementNONE, "None"},
		{oapi.ElementFIRE, "Fire"},
		{oapi.ElementTHUNDER, "Thunder"},
		{oapi.ElementCHILL, "Ice"},
		{oapi.ElementPHOTON, "Light"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ElementName(tt.et))
		})
	}
}

func TestElementName_InvalidPanic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = ElementName(oapi.Element("INVALID"))
	})
}
