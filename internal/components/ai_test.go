package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisposition_ResetToDefault(t *testing.T) {
	t.Parallel()

	d := &Disposition{Default: DispositionNeutral, Current: DispositionHostile}
	d.ResetToDefault()
	assert.Equal(t, DispositionNeutral, d.Current)
}

func TestDisposition_ReactToHostile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		defaultDisp     DispositionType
		expectedCurrent DispositionType
	}{
		{"NeutralはHostileになる", DispositionNeutral, DispositionHostile},
		{"CowardlyはFleeingになる", DispositionCowardly, DispositionFleeing},
		{"Hostileはそのまま", DispositionHostile, DispositionHostile},
		{"Fleeingはそのまま", DispositionFleeing, DispositionFleeing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := &Disposition{Default: tt.defaultDisp, Current: tt.defaultDisp}
			d.ReactToHostile()
			assert.Equal(t, tt.expectedCurrent, d.Current)
		})
	}
}
