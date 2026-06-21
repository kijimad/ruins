package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeaponDamageKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   SkillID
		want ModifierKey
	}{
		{SkillSword, ModSwordDamage},
		{SkillSpear, ModSpearDamage},
		{SkillFist, ModFistDamage},
		{SkillBow, ModBowDamage},
		{SkillHandgun, ModHandgunDamage},
		{SkillRifle, ModRifleDamage},
		{SkillCannon, ModCannonDamage},
	}

	for _, tt := range tests {
		t.Run(string(tt.id), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WeaponDamageKey(tt.id))
		})
	}
}

func TestWeaponDamageKey_InvalidID(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		WeaponDamageKey("invalid")
	})
}

func TestWeaponAccuracyKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   SkillID
		want ModifierKey
	}{
		{SkillSword, ModSwordAccuracy},
		{SkillSpear, ModSpearAccuracy},
		{SkillFist, ModFistAccuracy},
		{SkillBow, ModBowAccuracy},
		{SkillHandgun, ModHandgunAccuracy},
		{SkillRifle, ModRifleAccuracy},
		{SkillCannon, ModCannonAccuracy},
	}

	for _, tt := range tests {
		t.Run(string(tt.id), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, WeaponAccuracyKey(tt.id))
		})
	}
}

func TestWeaponAccuracyKey_InvalidID(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		WeaponAccuracyKey("invalid")
	})
}

func TestElementResistKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		elem ElementType
		want ModifierKey
	}{
		{ElementTypeFire, ModFireResist},
		{ElementTypeThunder, ModThunderResist},
		{ElementTypeChill, ModChillResist},
		{ElementTypePhoton, ModPhotonResist},
	}

	for _, tt := range tests {
		t.Run(string(tt.elem), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ElementResistKey(tt.elem))
		})
	}
}

func TestElementResistKey_InvalidType(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		ElementResistKey("invalid")
	})
}
