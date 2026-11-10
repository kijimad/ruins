package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestWeaponDamageKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   oapi.SkillId
		want oapi.ProficiencyId
	}{
		{oapi.SkillIdSword, oapi.ProficiencyIdSwordDamage},
		{oapi.SkillIdSpear, oapi.ProficiencyIdSpearDamage},
		{oapi.SkillIdFist, oapi.ProficiencyIdFistDamage},
		{oapi.SkillIdBow, oapi.ProficiencyIdBowDamage},
		{oapi.SkillIdHandgun, oapi.ProficiencyIdHandgunDamage},
		{oapi.SkillIdRifle, oapi.ProficiencyIdRifleDamage},
		{oapi.SkillIdCannon, oapi.ProficiencyIdCannonDamage},
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
		id   oapi.SkillId
		want oapi.ProficiencyId
	}{
		{oapi.SkillIdSword, oapi.ProficiencyIdSwordAccuracy},
		{oapi.SkillIdSpear, oapi.ProficiencyIdSpearAccuracy},
		{oapi.SkillIdFist, oapi.ProficiencyIdFistAccuracy},
		{oapi.SkillIdBow, oapi.ProficiencyIdBowAccuracy},
		{oapi.SkillIdHandgun, oapi.ProficiencyIdHandgunAccuracy},
		{oapi.SkillIdRifle, oapi.ProficiencyIdRifleAccuracy},
		{oapi.SkillIdCannon, oapi.ProficiencyIdCannonAccuracy},
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
		want oapi.ProficiencyId
	}{
		{ElementTypeFire, oapi.ProficiencyIdFireResist},
		{ElementTypeThunder, oapi.ProficiencyIdThunderResist},
		{ElementTypeChill, oapi.ProficiencyIdChillResist},
		{ElementTypePhoton, oapi.ProficiencyIdPhotonResist},
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
