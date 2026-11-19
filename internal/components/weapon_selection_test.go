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
	for _, s := range []EquipmentSlotNumber{-1, SlotJewelry, SlotWeapon5 + 1} {
		assert.Error(t, WeaponSelection{Slot: s}.Validate(), s)
	}
}

func TestEquipmentSlotNumber_WeaponIndex(t *testing.T) {
	t.Parallel()

	got := map[EquipmentSlotNumber]int{}
	for _, s := range []EquipmentSlotNumber{SlotWeapon1, SlotWeapon3, SlotWeapon5} {
		i, ok := s.WeaponIndex()
		assert.True(t, ok, s)
		got[s] = i
	}
	assert.Equal(t, map[EquipmentSlotNumber]int{SlotWeapon1: 0, SlotWeapon3: 2, SlotWeapon5: 4}, got)
	for _, s := range []EquipmentSlotNumber{-1, SlotJewelry, SlotWeapon5 + 1} {
		_, ok := s.WeaponIndex()
		assert.False(t, ok, s)
	}
}
