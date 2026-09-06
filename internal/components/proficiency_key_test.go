package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWeaponDamageKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   SkillID
		want ProficiencyKey
	}{
		{SkillSword, ProfSwordDamage},
		{SkillSpear, ProfSpearDamage},
		{SkillFist, ProfFistDamage},
		{SkillBow, ProfBowDamage},
		{SkillHandgun, ProfHandgunDamage},
		{SkillRifle, ProfRifleDamage},
		{SkillCannon, ProfCannonDamage},
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
		want ProficiencyKey
	}{
		{SkillSword, ProfSwordAccuracy},
		{SkillSpear, ProfSpearAccuracy},
		{SkillFist, ProfFistAccuracy},
		{SkillBow, ProfBowAccuracy},
		{SkillHandgun, ProfHandgunAccuracy},
		{SkillRifle, ProfRifleAccuracy},
		{SkillCannon, ProfCannonAccuracy},
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
		want ProficiencyKey
	}{
		{ElementTypeFire, ProfFireResist},
		{ElementTypeThunder, ProfThunderResist},
		{ElementTypeChill, ProfChillResist},
		{ElementTypePhoton, ProfPhotonResist},
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
