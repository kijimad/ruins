package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAIPolicy_ResetCombat(t *testing.T) {
	t.Parallel()

	p := &AIPolicy{CombatDefault: CombatIgnore, CombatCurrent: CombatAttack}
	p.ResetCombat()
	assert.Equal(t, CombatIgnore, p.CombatCurrent)
}

func TestAIPolicy_ReactToHostile(t *testing.T) {
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
			p := &AIPolicy{CombatDefault: tt.defaultCombat, CombatCurrent: tt.defaultCombat}
			p.ReactToHostile()
			assert.Equal(t, tt.expectedCurrent, p.CombatCurrent)
		})
	}
}
