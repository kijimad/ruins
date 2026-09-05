package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestCalcModifierValue_AllSkillsZero(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	value := func(key ModifierKey) int { return int(CalcModifierValue(skills, nil, nil, key)) }

	// スキル値0のとき全倍率は100（等倍）
	for _, id := range WeaponSkillIDs {
		assert.Equal(t, 100, value(WeaponDamageKey(id)), "武器ダメージ %s は100", id)
		assert.Equal(t, 100, value(WeaponAccuracyKey(id)), "武器命中 %s は100", id)
	}
	assert.Equal(t, 100, value(ModColdProgress))
	assert.Equal(t, 100, value(ModHungerProgress))
	assert.Equal(t, 100, value(ModHealingEffect))
	assert.Equal(t, 100, value(ModMaxWeight))
	assert.Equal(t, 100, value(ModEnemyVision))
	assert.Equal(t, 100, value(ModMoveCost))
	assert.Equal(t, 100, value(ModCraftCost))
	assert.Equal(t, 100, value(ModSmithQuality))
	assert.Equal(t, 100, value(ModBuyPrice))
	assert.Equal(t, 100, value(ModSellPrice))
	assert.Equal(t, 100, value(ModHeavyArmor))
}

func TestCalcModifierValue_SkillEffects(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 2

	// 刀剣Lv2: ダメージ倍率 = 100 + 2*5 = 110
	assert.Equal(t, 110, int(CalcModifierValue(skills, nil, nil, ModSwordDamage)))
	// 刀剣Lv2: 命中倍率 = 100 + 2*3 = 106
	assert.Equal(t, 106, int(CalcModifierValue(skills, nil, nil, ModSwordAccuracy)))
	// 他の武器は影響なし
	assert.Equal(t, 100, int(CalcModifierValue(skills, nil, nil, ModSpearDamage)))
}

func TestCalcModifierValue_NegativeCoefficient(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillColdResist).Value = 3

	// 耐寒Lv3: 低体温進行 = 100 + 3*(-3) = 91
	assert.Equal(t, 91, int(CalcModifierValue(skills, nil, nil, ModColdProgress)))
	// 耐寒Lv3: 火耐性 = 100 + 0*(-3) = 100（SkillFireResistはLv0のまま）
	assert.Equal(t, 100, int(CalcModifierValue(skills, nil, nil, ModFireResist)))
}

func TestCalcModifierValue_WithAbilities(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 2

	abils := &Abilities{
		Strength: Ability{Total: 10},
	}

	// 刀剣Lv2 + STR10: ダメージ = 100 + 2*5 + 10*1 = 120
	assert.Equal(t, 120, int(CalcModifierValue(skills, abils, nil, ModSwordDamage)))
	// 刀剣Lv2 + STR10: 命中 = 100 + 2*3 + 10*1 = 116
	assert.Equal(t, 116, int(CalcModifierValue(skills, abils, nil, ModSwordAccuracy)))
}

func TestCalcModifierValue_AbilityNegativeDirection(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillColdResist).Value = 1

	abils := &Abilities{
		Vitality: Ability{Total: 5},
	}

	// 耐寒Lv1 + VIT5: 低体温進行 = 100 + 1*(-3) + 5*(-1) = 92
	assert.Equal(t, 92, int(CalcModifierValue(skills, abils, nil, ModColdProgress)))
}

func TestCalcModifierSources(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 3

	abils := &Abilities{
		Strength: Ability{Total: 8},
	}

	sources := CalcModifierSources(skills, abils, nil, ModSwordDamage)
	assert.Len(t, sources, 2, "スキルと能力値の2つのソースがある")
	assert.Equal(t, ModifierSource{Kind: SourceSkill, Skill: SkillSword, Amount: 3, Value: 15}, sources[0]) // 3*5
	assert.Equal(t, ModifierSource{Kind: SourceAbility, Ability: AblSTR, Amount: 8, Value: 8}, sources[1])  // 8*1
}

func TestCalcModifierValue_HealthPenalty(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeverityMedium,
	})

	// 不調は MoveCost へ直接足さず身体機能 Capacities に一本化する
	assert.Equal(t, 100, int(CalcModifierValue(skills, nil, hs, ModMoveCost)), "低体温は MoveCost へ足さない")
	// 中度の全身性低体温 6/10: 痛み6*2=12、意識=100-20-12/2=74。
	// 局所低下は無いので操作・歩行・視覚はいずれも意識乗数だけを受けて74
	assert.Equal(t, BodyCapacities{Pain: 12, Blood: 100, Consciousness: 74, Manipulation: 74, Moving: 74, Sight: 74}, hs.Capacities())
}

func TestCalcModifierValue_UnknownKey(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	// 未定義キーは等倍・内訳なし
	assert.Equal(t, 100, int(CalcModifierValue(skills, nil, nil, "unknown")))
	assert.Empty(t, CalcModifierSources(skills, nil, nil, "unknown"))
}

func TestCalcModifierValue_Negotiation(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillNegotiation).Value = 4

	// 交渉Lv4: 買値 = 100 + 4*(-2) = 92 (安く買える)
	assert.Equal(t, 92, int(CalcModifierValue(skills, nil, nil, ModBuyPrice)))
	// 交渉Lv4: 売値 = 100 + 4*2 = 108 (高く売れる)
	assert.Equal(t, 108, int(CalcModifierValue(skills, nil, nil, ModSellPrice)))
}

func TestCalcModifierValue_MultipleSkills(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 3
	skills.Get(SkillHandgun).Value = 5
	skills.Get(SkillCrafting).Value = 2

	abils := &Abilities{
		Strength:  Ability{Total: 8},
		Sensation: Ability{Total: 6},
		Dexterity: Ability{Total: 4},
	}

	// 刀剣Lv3 + STR8: ダメージ = 100 + 3*5 + 8*1 = 123
	assert.Equal(t, 123, int(CalcModifierValue(skills, abils, nil, ModSwordDamage)))
	// 拳銃Lv5 + SEN6: ダメージ = 100 + 5*5 + 6*1 = 131
	assert.Equal(t, 131, int(CalcModifierValue(skills, abils, nil, ModHandgunDamage)))
	// クラフトLv2 + DEX4: 素材消費 = 100 + 2*(-3) + 4*(-1) = 90
	assert.Equal(t, 90, int(CalcModifierValue(skills, abils, nil, ModCraftCost)))
	// 長物は未使用: ダメージ = 100 + 0*5 + 8*1 = 108（STR能力値のみ）
	assert.Equal(t, 108, int(CalcModifierValue(skills, abils, nil, ModSpearDamage)))
}

func TestCalcModifierValue_AllFactors(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSprinting).Value = 4

	abils := &Abilities{
		Agility: Ability{Total: 10},
	}

	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeveritySevere,
	})

	// 走破Lv4 + AGI10: MoveCost = 100 + 4*(-2) + 10*(-1) = 82。低体温は MoveCost へ足さない
	assert.Equal(t, 82, int(CalcModifierValue(skills, abils, hs, ModMoveCost)))
	// 重度の全身性低体温 6/10 は身体機能へ効く。意識=100-30-18/2=61、歩行=100*61/100=61
	assert.Equal(t, 61, int(hs.Capacities().Moving))

	// Sourcesはスキルと能力値の2要因。健康は Capacities 側なので MoveCost には載らない
	sources := CalcModifierSources(skills, abils, hs, ModMoveCost)
	assert.Len(t, sources, 2, "スキルと能力値の2つのソース")
}

func TestCalcModifierValue_FireAbility(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillRifle).Value = 4

	abils := &Abilities{
		Sensation: Ability{Total: 12},
	}

	// 小銃Lv4 + SEN12: ダメージ = 100 + 4*5 + 12*1 = 132
	assert.Equal(t, 132, int(CalcModifierValue(skills, abils, nil, ModRifleDamage)))
	// 小銃Lv4 + SEN12: 命中 = 100 + 4*3 + 12*1 = 124
	assert.Equal(t, 124, int(CalcModifierValue(skills, abils, nil, ModRifleAccuracy)))
}

func TestCalcModifierValue_AccuracyFoldsCapacity(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeverityMedium,
	})

	// 中度の全身性低体温で操作・視覚は74。スキルLv0の基礎命中100×74%=74
	assert.Equal(t, 74, int(CalcModifierValue(skills, nil, hs, ModSwordAccuracy)), "近接は操作機能を畳み込む")
	assert.Equal(t, 74, int(CalcModifierValue(skills, nil, hs, ModBowAccuracy)), "遠隔は視覚機能を畳み込む")

	// 内訳の末尾に身体機能の加法差分が載る。100→74 なので -26
	swordSrc := CalcModifierSources(skills, nil, hs, ModSwordAccuracy)
	assert.Equal(t, ModifierSource{Kind: SourceCapacity, Capacity: CapacityManipulation, Amount: 74, Value: -26}, swordSrc[len(swordSrc)-1])
	bowSrc := CalcModifierSources(skills, nil, hs, ModBowAccuracy)
	assert.Equal(t, ModifierSource{Kind: SourceCapacity, Capacity: CapacitySight, Amount: 74, Value: -26}, bowSrc[len(bowSrc)-1])
}

func TestCalcModifier_値は基準と内訳の和に一致する(t *testing.T) {
	t.Parallel()

	richSkills := NewSkills()
	richSkills.Get(SkillSword).Value = 3
	richSkills.Get(SkillBow).Value = 2
	richSkills.Get(SkillColdResist).Value = 4
	richSkills.Get(SkillNegotiation).Value = 5
	richSkills.Get(SkillStealth).Value = 6

	abils := &Abilities{
		Strength:  Ability{Total: 8},
		Sensation: Ability{Total: 6},
		Agility:   Ability{Total: 4},
		Dexterity: Ability{Total: 2},
		Vitality:  Ability{Total: 1},
	}

	sickHS := &HealthStatus{}
	sickHS.Parts[BodyPartWholeBody].SetCondition(HealthCondition{Type: ConditionHypothermia, Severity: SeverityMedium})
	sickHS.Parts[BodyPartHead].SetCondition(HealthCondition{Type: ConditionLaceration, Timer: 60, Severity: TimerToSeverity(60)})

	cases := []struct {
		name   string
		skills *Skills
		abils  *Abilities
		hs     *HealthStatus
	}{
		{"素の状態", NewSkills(), nil, nil},
		{"スキルと能力値", richSkills, abils, nil},
		{"不調あり", richSkills, abils, sickHS},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// 表示は 基準 + Σ内訳、適用は CalcModifierValue。両者の一致が表示=適用の土台
			for _, spec := range modifierSpecs {
				sum := int(consts.PercentBase)
				for _, s := range CalcModifierSources(tc.skills, tc.abils, tc.hs, spec.Key) {
					sum += s.Value
				}
				assert.Equal(t, int(CalcModifierValue(tc.skills, tc.abils, tc.hs, spec.Key)), sum,
					"キー %s で 最終値 = 基準 + Σ内訳 が破れた", spec.Key)
			}
		})
	}
}

func TestCalcModifierValue_ElementResistAllTypes(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillFireResist).Value = 2
	skills.Get(SkillThunderResist).Value = 4
	skills.Get(SkillChillResist).Value = 6
	skills.Get(SkillPhotonResist).Value = 8

	// 各元素耐性: 100 + Lv*(-3)
	assert.Equal(t, 94, int(CalcModifierValue(skills, nil, nil, ModFireResist)))    // 100 + 2*(-3)
	assert.Equal(t, 88, int(CalcModifierValue(skills, nil, nil, ModThunderResist))) // 100 + 4*(-3)
	assert.Equal(t, 82, int(CalcModifierValue(skills, nil, nil, ModChillResist)))   // 100 + 6*(-3)
	assert.Equal(t, 76, int(CalcModifierValue(skills, nil, nil, ModPhotonResist)))  // 100 + 8*(-3)
}
