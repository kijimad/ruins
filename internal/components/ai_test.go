package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSoloAI_ResetCombat(t *testing.T) {
	t.Parallel()

	solo := &SoloAI{CombatDefault: CombatIgnore, CombatCurrent: CombatAttack}
	solo.ResetCombat()
	assert.Equal(t, CombatIgnore, solo.CombatCurrent)
}

func TestSoloAI_ReactToHostile(t *testing.T) {
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
			solo := &SoloAI{CombatDefault: tt.defaultCombat, CombatCurrent: tt.defaultCombat}
			solo.ReactToHostile()
			assert.Equal(t, tt.expectedCurrent, solo.CombatCurrent)
		})
	}
}

func TestCombatPolicy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    CombatPolicy
		want string
	}{
		{CombatAttack, "Attack"},
		{CombatEvade, "Evade"},
		{CombatIgnore, "Indifferent"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}

func TestSoloMovement_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    SoloMovement
		want string
	}{
		{SoloRandom, "Random"},
		{SoloPatrol, "Patrol"},
		{SoloWallHug, "Along walls"},
		{SoloStationary, "Fixed"},
		{SoloWander, "Wander"},
		{SoloTerritorial, "Territory"},
		{SoloSwarm, "Flock"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}
