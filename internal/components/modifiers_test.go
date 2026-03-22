package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecalculateCharModifiers_AllSkillsZero(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	mods := RecalculateCharModifiers(skills, nil, nil)

	// スキル値0のとき全倍率は100（等倍）
	for _, id := range weaponSkillIDs {
		assert.Equal(t, 100, mods.WeaponDamage[id], "武器ダメージ %s は100", id)
		assert.Equal(t, 100, mods.WeaponAccuracy[id], "武器命中 %s は100", id)
	}
	assert.Equal(t, 100, mods.ColdProgress)
	assert.Equal(t, 100, mods.HeatProgress)
	assert.Equal(t, 100, mods.HungerProgress)
	assert.Equal(t, 100, mods.HealingEffect)
	assert.Equal(t, 100, mods.MaxWeight)
	assert.Equal(t, 100, mods.EnemyVision)
	assert.Equal(t, 100, mods.MoveCost)
	assert.Equal(t, 100, mods.CraftCost)
	assert.Equal(t, 100, mods.SmithQuality)
	assert.Equal(t, 100, mods.BuyPrice)
	assert.Equal(t, 100, mods.SellPrice)
	assert.Equal(t, 100, mods.HeavyArmor)
}

func TestRecalculateCharModifiers_SkillEffects(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Data[SkillSword].Value = 2

	mods := RecalculateCharModifiers(skills, nil, nil)

	// 刀剣Lv2: ダメージ倍率 = 100 + 2*5 = 110
	assert.Equal(t, 110, mods.WeaponDamage[SkillSword])
	// 刀剣Lv2: 命中倍率 = 100 + 2*3 = 106
	assert.Equal(t, 106, mods.WeaponAccuracy[SkillSword])
	// 他の武器は影響なし
	assert.Equal(t, 100, mods.WeaponDamage[SkillSpear])
}

func TestRecalculateCharModifiers_NegativeCoefficient(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Data[SkillColdResist].Value = 3

	mods := RecalculateCharModifiers(skills, nil, nil)

	// 耐寒Lv3: 低体温進行 = 100 + 3*(-3) = 91
	assert.Equal(t, 91, mods.ColdProgress)
	// 耐寒Lv3: 火耐性 = 100 + 0*(-3) = 100（SkillFireResistはLv0のまま）
	assert.Equal(t, 100, mods.ElementResist[ElementTypeFire])
}

func TestRecalculateCharModifiers_WithAbilities(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Data[SkillSword].Value = 2

	abils := &Abilities{
		Strength: Ability{Total: 10},
	}

	mods := RecalculateCharModifiers(skills, abils, nil)

	// 刀剣Lv2 + STR10: ダメージ = 100 + 2*5 + 10*1 = 120
	assert.Equal(t, 120, mods.WeaponDamage[SkillSword])
	// 刀剣Lv2 + STR10: 命中 = 100 + 2*3 + 10*1 = 116
	assert.Equal(t, 116, mods.WeaponAccuracy[SkillSword])
}

func TestRecalculateCharModifiers_AbilityNegativeDirection(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Data[SkillColdResist].Value = 1

	abils := &Abilities{
		Vitality: Ability{Total: 5},
	}

	mods := RecalculateCharModifiers(skills, abils, nil)

	// 耐寒Lv1 + VIT5: 低体温進行 = 100 + 1*(-3) + 5*(-1) = 92
	assert.Equal(t, 92, mods.ColdProgress)
}

func TestRecalculateCharModifiers_Sources(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Data[SkillSword].Value = 3

	abils := &Abilities{
		Strength: Ability{Total: 8},
	}

	mods := RecalculateCharModifiers(skills, abils, nil)

	sources := mods.Sources[ModSwordDamage]
	assert.Len(t, sources, 2, "スキルと能力値の2つのソースがある")
	assert.Equal(t, "刀剣 Lv3", sources[0].Label)
	assert.Equal(t, 15, sources[0].Value) // 3*5
	assert.Equal(t, "STR 8", sources[1].Label)
	assert.Equal(t, 8, sources[1].Value) // 8*1
}

func TestRecalculateCharModifiers_HealthPenalty(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeverityMedium,
	})

	mods := RecalculateCharModifiers(skills, nil, hs)

	// 中度低体温: MoveCost = 100 + 20
	assert.Equal(t, 120, mods.MoveCost)

	// Sourcesに低体温のペナルティが記録される
	sources := mods.Sources[ModMoveCost]
	found := false
	for _, s := range sources {
		if s.Label == "低体温" {
			assert.Equal(t, 20, s.Value)
			found = true
		}
	}
	assert.True(t, found, "MoveCostのSourcesに低体温が含まれる")
}

func TestTemperatureMovePenalty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		severity Severity
		expected int
	}{
		{SeverityNone, 0},
		{SeverityMinor, 10},
		{SeverityMedium, 20},
		{SeveritySevere, 30},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, temperatureMovePenalty(tt.severity))
	}
}

func TestRecalculateCharModifiers_NilAbilsAndHS(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	// panicしないことを確認
	mods := RecalculateCharModifiers(skills, nil, nil)
	assert.NotNil(t, mods)
}

func TestRecalculateCharModifiers_Negotiation(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Data[SkillNegotiation].Value = 4

	mods := RecalculateCharModifiers(skills, nil, nil)

	// 交渉Lv4: 買値 = 100 + 4*(-2) = 92 (安く買える)
	assert.Equal(t, 92, mods.BuyPrice)
	// 交渉Lv4: 売値 = 100 + 4*2 = 108 (高く売れる)
	assert.Equal(t, 108, mods.SellPrice)
}
