package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeaponSelection_Validate(t *testing.T) {
	t.Parallel()

	for _, s := range []EquipmentSlotNumber{SlotHand1, SlotHand5} {
		require.NoError(t, WeaponSelection{Slot: s}.Validate(), s)
	}
	for _, s := range []EquipmentSlotNumber{-1, SlotJewelry, SlotHand5 + 1} {
		assert.Error(t, WeaponSelection{Slot: s}.Validate(), s)
	}
}

func TestEquipmentSlotNumber_WeaponIndex(t *testing.T) {
	t.Parallel()

	got := map[EquipmentSlotNumber]int{}
	for _, s := range []EquipmentSlotNumber{SlotHand1, SlotHand3, SlotHand5} {
		i, ok := s.HandIndex()
		assert.True(t, ok, s)
		got[s] = i
	}
	assert.Equal(t, map[EquipmentSlotNumber]int{SlotHand1: 0, SlotHand3: 2, SlotHand5: 4}, got)
	for _, s := range []EquipmentSlotNumber{-1, SlotJewelry, SlotHand5 + 1} {
		_, ok := s.HandIndex()
		assert.False(t, ok, s)
	}
}
