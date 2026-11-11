package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestSoloAI_ResetCombat(t *testing.T) {
	t.Parallel()

	solo := &SoloAI{CombatDefault: rawschema.Ignore, CombatCurrent: rawschema.Attack}
	solo.ResetCombat()
	assert.Equal(t, rawschema.Ignore, solo.CombatCurrent)
}

func TestSoloAI_ReactToHostile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		defaultCombat   rawschema.CombatPolicyType
		expectedCurrent rawschema.CombatPolicyType
	}{
		{"CombatIgnoreはCombatAttackになる", rawschema.Ignore, rawschema.Attack},
		{"CombatAttackはそのまま", rawschema.Attack, rawschema.Attack},
		{"CombatEvadeはそのまま", rawschema.Evade, rawschema.Evade},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			solo := &SoloAI{CombatDefault: tt.defaultCombat, CombatCurrent: tt.defaultCombat}
			solo.ReactToHostile()
			assert.Equal(t, tt.expectedCurrent, solo.CombatCurrent)
		})
	}
}
