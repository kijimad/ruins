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

func TestPlannerType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    PlannerType
		want string
	}{
		{PlannerRoaming, "徘徊"},
		{PlannerSquad, "隊員"},
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
		{CombatAttack, "攻撃"},
		{CombatEvade, "回避"},
		{CombatIgnore, "無関心"},
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

func TestMovementPolicy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    MovementPolicy
		want string
	}{
		{MovementEscort, "護衛"},
		{MovementVanguard, "前衛"},
		{MovementPatrol, "巡回"},
		{MovementStationary, "固定"},
		{MovementRetreat, "後退"},
		{MovementRandom, "ランダム"},
		{MovementWallHug, "壁沿い"},
		{MovementWander, "徘徊"},
		{MovementTerritorial, "縄張り"},
		{MovementSwarm, "群れ"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}

func TestAllSquadMovementPolicies(t *testing.T) {
	t.Parallel()

	policies := AllSquadMovementPolicies()
	assert.Len(t, policies, 5)
	assert.Contains(t, policies, MovementEscort)
	assert.Contains(t, policies, MovementVanguard)
	assert.Contains(t, policies, MovementPatrol)
	assert.Contains(t, policies, MovementStationary)
	assert.Contains(t, policies, MovementRetreat)
}

func TestItemPickupPolicy_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		p    ItemPickupPolicy
		want string
	}{
		{PolicyPickup, "回収"},
		{PolicyIgnore, "無視"},
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
		{PolicyKeep, "保持"},
		{PolicyDistribute, "分配"},
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
