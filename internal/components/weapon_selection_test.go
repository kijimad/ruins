package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeaponSelection_Validate(t *testing.T) {
	t.Parallel()

	for _, s := range []EquipmentSlotNumber{SlotWeapon1, SlotWeapon5} {
		require.NoError(t, WeaponSelection{Slot: s}.Validate(), s)
	}
	for _, s := range []EquipmentSlotNumber{SlotJewelry, SlotWeapon5 + 1} {
		assert.Error(t, WeaponSelection{Slot: s}.Validate(), s)
	}
}
