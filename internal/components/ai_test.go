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

func TestPlannerType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    PlannerType
		want string
	}{
		{PlannerSolo, "Solo"},
		{PlannerSquad, "Squad member"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.String())
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

func TestAllSquadCombatPolicies(t *testing.T) {
	t.Parallel()

	policies := AllSquadCombatPolicies()
	assert.Equal(t, []CombatPolicy{CombatAttack, CombatEvade}, policies)
	assert.NotContains(t, policies, CombatIgnore, "CombatIgnoreは隊員用ではない")
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

func TestSquadMovement_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    SquadMovement
		want string
	}{
		{SquadEscort, "Escort"},
		{SquadVanguard, "Vanguard"},
		{SquadPatrol, "Patrol"},
		{SquadStationary, "Fixed"},
		{SquadRetreat, "Retreat"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}

func TestAllSquadMovements(t *testing.T) {
	t.Parallel()

	policies := AllSquadMovements()
	assert.Len(t, policies, 5)
	assert.Contains(t, policies, SquadEscort)
	assert.Contains(t, policies, SquadVanguard)
	assert.Contains(t, policies, SquadPatrol)
	assert.Contains(t, policies, SquadStationary)
	assert.Contains(t, policies, SquadRetreat)
}

func TestItemPickupPolicy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    ItemPickupPolicy
		want string
	}{
		{PolicyPickup, "Collect"},
		{PolicyIgnore, "Ignore"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}

func TestAllItemPickupPolicies(t *testing.T) {
	t.Parallel()

	policies := AllItemPickupPolicies()
	assert.Equal(t, []ItemPickupPolicy{PolicyPickup, PolicyIgnore}, policies)
}

func TestItemHandlingPolicy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    ItemHandlingPolicy
		want string
	}{
		{PolicyKeep, "Hold"},
		{PolicyDistribute, "Distribute"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}

func TestAllItemHandlingPolicies(t *testing.T) {
	t.Parallel()

	policies := AllItemHandlingPolicies()
	assert.Equal(t, []ItemHandlingPolicy{PolicyKeep, PolicyDistribute}, policies)
}
