package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAI_ResetCombat(t *testing.T) {
	t.Parallel()

	ai := &AI{CombatDefault: CombatIgnore, CombatCurrent: CombatAttack}
	ai.ResetCombat()
	assert.Equal(t, CombatIgnore, ai.CombatCurrent)
}

func TestAI_ReactToHostile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		defaultCombat   CombatPolicy
		expectedCurrent CombatPolicy
	}{
		{"CombatIgnoreはCombatAttackになる", CombatIgnore, CombatAttack},
		{"CombatAttackはそのまま", CombatAttack, CombatAttack},
		{"CombatEvadeはそのまま", CombatEvade, CombatEvade},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ai := &AI{CombatDefault: tt.defaultCombat, CombatCurrent: tt.defaultCombat}
			ai.ReactToHostile()
			assert.Equal(t, tt.expectedCurrent, ai.CombatCurrent)
		})
	}
}
