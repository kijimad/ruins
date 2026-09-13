package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultParams_倍率つまみは1(t *testing.T) {
	t.Parallel()
	p := DefaultParams()
	assert.Equal(t, 1.0, p.EnemyWeaponScale)
	assert.Equal(t, 1.0, p.PlayerStrengthScale)
	assert.Equal(t, 1.0, p.FuelHeatScale)
	assert.Equal(t, 1.0, p.LootValueScale)
}

func TestKnobRegistry_名前が一意で別成分を指す(t *testing.T) {
	t.Parallel()
	knobs := knobRegistry()
	names := make(map[string]bool, len(knobs))
	ptrs := make(map[*float64]bool, len(knobs))
	var p Params
	for _, k := range knobs {
		assert.False(t, names[k.Name], "つまみ名の重複: "+k.Name)
		names[k.Name] = true
		ptr := k.Ptr(&p)
		assert.False(t, ptrs[ptr], "同じ成分を指すつまみ: "+k.Name)
		ptrs[ptr] = true
	}
}
