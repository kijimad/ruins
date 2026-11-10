package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestSoloAI_ResetCombat(t *testing.T) {
	t.Parallel()

	solo := &SoloAI{CombatDefault: oapi.Ignore, CombatCurrent: oapi.Attack}
	solo.ResetCombat()
	assert.Equal(t, oapi.Ignore, solo.CombatCurrent)
}

func TestSoloAI_ReactToHostile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		defaultCombat   oapi.CombatPolicyType
		expectedCurrent oapi.CombatPolicyType
	}{
		{"CombatIgnoreはCombatAttackになる", oapi.Ignore, oapi.Attack},
		{"CombatAttackはそのまま", oapi.Attack, oapi.Attack},
		{"CombatEvadeはそのまま", oapi.Evade, oapi.Evade},
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
