package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestCalcProficiencyValue_AllSkillsZero(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	value := func(key ProficiencyKey) int {
		return int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), key))
	}

	// スキル値0のとき全倍率は100（等倍）
	for _, id := range WeaponSkillIDs {
		assert.Equal(t, 100, value(WeaponDamageKey(id)), "武器ダメージ %s は100", id)
		assert.Equal(t, 100, value(WeaponAccuracyKey(id)), "武器命中 %s は100", id)
	}
	assert.Equal(t, 100, value(ProfColdProgress))
	assert.Equal(t, 100, value(ProfHungerProgress))
	assert.Equal(t, 100, value(ProfHealingEffect))
	assert.Equal(t, 100, value(ProfMaxWeight))
	assert.Equal(t, 100, value(ProfEnemyVision))
	assert.Equal(t, 100, value(ProfMoveCost))
	assert.Equal(t, 100, value(ProfCraftCost))
	assert.Equal(t, 100, value(ProfSmithQuality))
	assert.Equal(t, 100, value(ProfBuyPrice))
	assert.Equal(t, 100, value(ProfSellPrice))
	assert.Equal(t, 100, value(ProfHeavyArmor))
}

func TestCalcProficiencyValue_SkillEffects(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 2

	// 刀剣Lv2: ダメージ倍率 = 100 + 2*5 = 110
	assert.Equal(t, 110, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfSwordDamage)))
	// 刀剣Lv2: 命中倍率 = 100 + 2*3 = 106
	assert.Equal(t, 106, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfSwordAccuracy)))
	// 他の武器は影響なし
	assert.Equal(t, 100, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfSpearDamage)))
}

func TestCalcProficiencyValue_NegativeCoefficient(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillColdResist).Value = 3

	// 耐寒Lv3: 低体温進行 = 100 + 3*(-3) = 91
	assert.Equal(t, 91, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfColdProgress)))
	// 耐寒Lv3: 火耐性 = 100 + 0*(-3) = 100（SkillFireResistはLv0のまま）
	assert.Equal(t, 100, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfFireResist)))
}

func TestCalcProficiencyValue_WithAbilities(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 2

	abils := &Abilities{
		Strength: Ability{Total: 10},
	}

	// 刀剣Lv2 + STR10: ダメージ = 100 + 2*5 + 10*1 = 120
	assert.Equal(t, 120, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfSwordDamage)))
	// 刀剣Lv2 + STR10: 命中 = 100 + 2*3 + 10*1 = 116
	assert.Equal(t, 116, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfSwordAccuracy)))
}

func TestCalcProficiencyValue_AbilityNegativeDirection(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillColdResist).Value = 1

	abils := &Abilities{
		Vitality: Ability{Total: 5},
	}

	// 耐寒Lv1 + VIT5: 低体温進行 = 100 + 1*(-3) + 5*(-1) = 92
	assert.Equal(t, 92, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfColdProgress)))
}

func TestCalcProficiencySources(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 3

	abils := &Abilities{
		Strength: Ability{Total: 8},
	}

	sources := CalcProficiencySources(skills, abils, HealthyBodyFuncs(), ProfSwordDamage)
	assert.Len(t, sources, 2, "スキルと能力値の2つのソースがある")
	assert.Equal(t, ProficiencySource{Kind: SourceSkill, Skill: SkillSword, Amount: 3, Value: 15}, sources[0]) // 3*5
	assert.Equal(t, ProficiencySource{Kind: SourceAbility, Ability: AblSTR, Amount: 8, Value: 8}, sources[1])  // 8*1
}

func TestCalcProficiencyValue_HealthPenalty(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeverityMedium,
	})

	// 不調は MoveCost へ直接足さず身体機能 BodyFuncs に一本化する
	assert.Equal(t, 100, int(CalcProficiencyValue(skills, nil, hs.BodyFuncs(), ProfMoveCost)), "低体温は MoveCost へ足さない")
	// 中度の全身性低体温 6/10: 痛み6*2=12、意識=100-20-12/2=74。
	// 局所低下は無いので操作・歩行・視覚はいずれも意識乗数だけを受けて74
	assert.Equal(t, BodyFuncs{Pain: 12, Blood: 100, Consciousness: 74, Manipulation: 74, Moving: 74, Sight: 74}, hs.BodyFuncs())
}

func TestCalcProficiencyValue_UnknownKey(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	// 未定義キーは等倍・内訳なし
	assert.Equal(t, 100, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), "unknown")))
	assert.Empty(t, CalcProficiencySources(skills, nil, HealthyBodyFuncs(), "unknown"))
}

func TestCalcProficiencyValue_Negotiation(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillNegotiation).Value = 4

	// 交渉Lv4: 買値 = 100 + 4*(-2) = 92 (安く買える)
	assert.Equal(t, 92, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfBuyPrice)))
	// 交渉Lv4: 売値 = 100 + 4*2 = 108 (高く売れる)
	assert.Equal(t, 108, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfSellPrice)))
}

func TestCalcProficiencyValue_MultipleSkills(t *testing.T) {
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
	assert.Equal(t, 123, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfSwordDamage)))
	// 拳銃Lv5 + SEN6: ダメージ = 100 + 5*5 + 6*1 = 131
	assert.Equal(t, 131, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfHandgunDamage)))
	// クラフトLv2 + DEX4: 素材消費 = 100 + 2*(-3) + 4*(-1) = 90
	assert.Equal(t, 90, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfCraftCost)))
	// 長物は未使用: ダメージ = 100 + 0*5 + 8*1 = 108（STR能力値のみ）
	assert.Equal(t, 108, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfSpearDamage)))
}

func TestCalcProficiencyValue_AllFactors(t *testing.T) {
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
	assert.Equal(t, 82, int(CalcProficiencyValue(skills, abils, hs.BodyFuncs(), ProfMoveCost)))
	// 重度の全身性低体温 6/10 は身体機能へ効く。意識=100-30-18/2=61、歩行=100*61/100=61
	assert.Equal(t, 61, int(hs.BodyFuncs().Moving))

	// Sourcesはスキルと能力値の2要因。健康は BodyFuncs 側なので MoveCost には載らない
	sources := CalcProficiencySources(skills, abils, hs.BodyFuncs(), ProfMoveCost)
	assert.Len(t, sources, 2, "スキルと能力値の2つのソース")
}

func TestCalcProficiencyValue_FireAbility(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillRifle).Value = 4

	abils := &Abilities{
		Sensation: Ability{Total: 12},
	}

	// 小銃Lv4 + SEN12: ダメージ = 100 + 4*5 + 12*1 = 132
	assert.Equal(t, 132, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfRifleDamage)))
	// 小銃Lv4 + SEN12: 命中 = 100 + 4*3 + 12*1 = 124
	assert.Equal(t, 124, int(CalcProficiencyValue(skills, abils, HealthyBodyFuncs(), ProfRifleAccuracy)))
}

func TestCalcProficiencyValue_AccuracyFoldsBodyFunc(t *testing.T) {
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
	assert.Equal(t, 74, int(CalcProficiencyValue(skills, nil, hs.BodyFuncs(), ProfSwordAccuracy)), "近接は操作機能を畳み込む")
	assert.Equal(t, 74, int(CalcProficiencyValue(skills, nil, hs.BodyFuncs(), ProfBowAccuracy)), "遠隔は視覚機能を畳み込む")

	// 内訳の末尾に身体機能の加法差分が載る。100→74 なので -26
	swordSrc := CalcProficiencySources(skills, nil, hs.BodyFuncs(), ProfSwordAccuracy)
	assert.Equal(t, ProficiencySource{Kind: SourceBodyFunc, BodyFunc: BodyFuncManipulation, Amount: 74, Value: -26}, swordSrc[len(swordSrc)-1])
	bowSrc := CalcProficiencySources(skills, nil, hs.BodyFuncs(), ProfBowAccuracy)
	assert.Equal(t, ProficiencySource{Kind: SourceBodyFunc, BodyFunc: BodyFuncSight, Amount: 74, Value: -26}, bowSrc[len(bowSrc)-1])
}

func TestCalcProficiency_値は基準と内訳の和に一致する(t *testing.T) {
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
		caps   BodyFuncs
	}{
		{"素の状態", NewSkills(), nil, HealthyBodyFuncs()},
		{"スキルと能力値", richSkills, abils, HealthyBodyFuncs()},
		{"不調あり", richSkills, abils, sickHS.BodyFuncs()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// 表示は 基準 + Σ内訳、適用は CalcProficiencyValue。両者の一致が表示=適用の土台
			for _, spec := range proficiencySpecs {
				sum := int(consts.PercentBase)
				for _, s := range CalcProficiencySources(tc.skills, tc.abils, tc.caps, spec.Key) {
					sum += s.Value
				}
				assert.Equal(t, int(CalcProficiencyValue(tc.skills, tc.abils, tc.caps, spec.Key)), sum,
					"キー %s で 最終値 = 基準 + Σ内訳 が破れた", spec.Key)
			}
		})
	}
}

func TestCalcProficiencyValue_ElementResistAllTypes(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillFireResist).Value = 2
	skills.Get(SkillThunderResist).Value = 4
	skills.Get(SkillChillResist).Value = 6
	skills.Get(SkillPhotonResist).Value = 8

	// 各元素耐性: 100 + Lv*(-3)
	assert.Equal(t, 94, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfFireResist)))    // 100 + 2*(-3)
	assert.Equal(t, 88, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfThunderResist))) // 100 + 4*(-3)
	assert.Equal(t, 82, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfChillResist)))   // 100 + 6*(-3)
	assert.Equal(t, 76, int(CalcProficiencyValue(skills, nil, HealthyBodyFuncs(), ProfPhotonResist)))  // 100 + 8*(-3)
}
