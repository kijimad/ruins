package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestSkillDefsProficiencyValue_AllSkillsZero(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	value := func(key oapi.ProficiencyId) int {
		return int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), key))
	}

	// スキル値0のとき全倍率は100（等倍）
	for _, w := range testSkillDefs.Weapons {
		assert.Equal(t, 100, value(w.Damage), "武器ダメージ %s は100", w.Damage)
		assert.Equal(t, 100, value(w.Accuracy), "武器命中 %s は100", w.Accuracy)
	}
	assert.Equal(t, 100, value(oapi.ProficiencyIdColdProgress))
	assert.Equal(t, 100, value(oapi.ProficiencyIdHungerProgress))
	assert.Equal(t, 100, value(oapi.ProficiencyIdHealingEffect))
	assert.Equal(t, 100, value(oapi.ProficiencyIdMaxWeight))
	assert.Equal(t, 100, value(oapi.ProficiencyIdEnemyVision))
	assert.Equal(t, 100, value(oapi.ProficiencyIdMoveCost))
	assert.Equal(t, 100, value(oapi.ProficiencyIdCraftCost))
	assert.Equal(t, 100, value(oapi.ProficiencyIdSmithQuality))
	assert.Equal(t, 100, value(oapi.ProficiencyIdBuyPrice))
	assert.Equal(t, 100, value(oapi.ProficiencyIdSellPrice))
	assert.Equal(t, 100, value(oapi.ProficiencyIdHeavyArmor))
}

func TestSkillDefsProficiencyValue_SkillEffects(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdSword).Value = 2

	// 刀剣Lv2: ダメージ倍率 = 100 + 2*5 = 110
	assert.Equal(t, 110, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdSwordDamage)))
	// 刀剣Lv2: 命中倍率 = 100 + 2*3 = 106
	assert.Equal(t, 106, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdSwordAccuracy)))
	// 他の武器は影響なし
	assert.Equal(t, 100, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdSpearDamage)))
}

func TestSkillDefsProficiencyValue_NegativeCoefficient(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdColdResist).Value = 3

	// 耐寒Lv3: 低体温進行 = 100 + 3*(-3) = 91
	assert.Equal(t, 91, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdColdProgress)))
	// 耐寒Lv3: 火耐性 = 100 + 0*(-3) = 100（SkillFireResistはLv0のまま）
	assert.Equal(t, 100, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdFireResist)))
}

func TestSkillDefsProficiencyValue_WithAbilities(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdSword).Value = 2

	abils := &Abilities{
		Strength: Ability{Total: 10},
	}

	// 刀剣Lv2 + STR10: ダメージ = 100 + 2*5 + 10*1 = 120
	assert.Equal(t, 120, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdSwordDamage)))
	// 刀剣Lv2 + STR10: 命中 = 100 + 2*3 + 10*1 = 116
	assert.Equal(t, 116, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdSwordAccuracy)))
}

func TestSkillDefsProficiencyValue_AbilityNegativeDirection(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdColdResist).Value = 1

	abils := &Abilities{
		Vitality: Ability{Total: 5},
	}

	// 耐寒Lv1 + VIT5: 低体温進行 = 100 + 1*(-3) + 5*(-1) = 92
	assert.Equal(t, 92, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdColdProgress)))
}

func TestSkillDefsProficiencySources(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdSword).Value = 3

	abils := &Abilities{
		Strength: Ability{Total: 8},
	}

	sources := testSkillDefs.ProficiencySources(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdSwordDamage)
	assert.Len(t, sources, 2, "スキルと能力値の2つのソースがある")
	assert.Equal(t, ProficiencySource{Kind: SourceSkill, Skill: oapi.SkillIdSword, Amount: 3, Value: 15}, sources[0]) // 3*5
	assert.Equal(t, ProficiencySource{Kind: SourceAbility, Ability: oapi.Str, Amount: 8, Value: 8}, sources[1])       // 8*1
}

func TestSkillDefsProficiencyValue_HealthPenalty(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionType("hypothermia"),
		Severity: SeverityMedium,
	})

	// 不調は MoveCost へ直接足さず身体機能 BodyFuncs に一本化する
	assert.Equal(t, 100, int(testSkillDefs.ProficiencyValue(skills, nil, hs.BodyFuncs(testDefs), oapi.ProficiencyIdMoveCost)), "低体温は MoveCost へ足さない")
	// 中度の全身性低体温 6/10: 痛み6*2=12、意識=100-20-12/2=74。
	// 意識は master 乗数。局所低下は無いので操作・歩行・視覚はいずれも意識だけを受けて74。代謝は過労・栄養失調なしで100
	assert.Equal(t, BodyFuncs{Pain: 12, Blood: 100, Consciousness: 74, Manipulation: 74, Moving: 74, Sight: 74, Metabolism: 100}, hs.BodyFuncs(testDefs))
}

func TestSkillDefsProficiencyValue_UnknownKey(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	// 未定義キーは等倍・内訳なし
	assert.Equal(t, 100, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), "unknown")))
	assert.Empty(t, testSkillDefs.ProficiencySources(skills, nil, HealthyBodyFuncs(), "unknown"))
}

func TestSkillDefsProficiencyValue_Negotiation(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdNegotiation).Value = 4

	// 交渉Lv4: 買値 = 100 + 4*(-2) = 92 (安く買える)
	assert.Equal(t, 92, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdBuyPrice)))
	// 交渉Lv4: 売値 = 100 + 4*2 = 108 (高く売れる)
	assert.Equal(t, 108, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdSellPrice)))
}

func TestSkillDefsProficiencyValue_MultipleSkills(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdSword).Value = 3
	skills.Get(oapi.SkillIdHandgun).Value = 5
	skills.Get(oapi.SkillIdCrafting).Value = 2

	abils := &Abilities{
		Strength:  Ability{Total: 8},
		Sensation: Ability{Total: 6},
		Dexterity: Ability{Total: 4},
	}

	// 刀剣Lv3 + STR8: ダメージ = 100 + 3*5 + 8*1 = 123
	assert.Equal(t, 123, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdSwordDamage)))
	// 拳銃Lv5 + SEN6: ダメージ = 100 + 5*5 + 6*1 = 131
	assert.Equal(t, 131, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdHandgunDamage)))
	// クラフトLv2 + DEX4: 素材消費 = 100 + 2*(-3) + 4*(-1) = 90
	assert.Equal(t, 90, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdCraftCost)))
	// 長物は未使用: ダメージ = 100 + 0*5 + 8*1 = 108（STR能力値のみ）
	assert.Equal(t, 108, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdSpearDamage)))
}

func TestSkillDefsProficiencyValue_AllFactors(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdSprinting).Value = 4

	abils := &Abilities{
		Agility: Ability{Total: 10},
	}

	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionType("hypothermia"),
		Severity: SeveritySevere,
	})

	// 走破Lv4 + AGI10: MoveCost = 100 + 4*(-2) + 10*(-1) = 82。低体温は MoveCost へ足さない
	assert.Equal(t, 82, int(testSkillDefs.ProficiencyValue(skills, abils, hs.BodyFuncs(testDefs), oapi.ProficiencyIdMoveCost)))
	// 重度の全身性低体温 6/10 は意識を下げる。意識=100-30-18/2=61、歩行=100*61/100=61
	assert.Equal(t, 61, int(hs.BodyFuncs(testDefs).Consciousness))
	assert.Equal(t, 61, int(hs.BodyFuncs(testDefs).Moving))

	// Sourcesはスキルと能力値の2要因。健康は BodyFuncs 側なので MoveCost には載らない
	sources := testSkillDefs.ProficiencySources(skills, abils, hs.BodyFuncs(testDefs), oapi.ProficiencyIdMoveCost)
	assert.Len(t, sources, 2, "スキルと能力値の2つのソース")
}

func TestSkillDefsProficiencyValue_FireAbility(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdRifle).Value = 4

	abils := &Abilities{
		Sensation: Ability{Total: 12},
	}

	// 小銃Lv4 + SEN12: ダメージ = 100 + 4*5 + 12*1 = 132
	assert.Equal(t, 132, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdRifleDamage)))
	// 小銃Lv4 + SEN12: 命中 = 100 + 4*3 + 12*1 = 124
	assert.Equal(t, 124, int(testSkillDefs.ProficiencyValue(skills, abils, HealthyBodyFuncs(), oapi.ProficiencyIdRifleAccuracy)))
}

func TestSkillDefsProficiencyValue_AccuracyFoldsBodyFunc(t *testing.T) {
	t.Parallel()

	hasBodyFunc := func(srcs []ProficiencySource, bf BodyFuncKind) bool {
		for _, s := range srcs {
			if s.Kind == SourceBodyFunc && s.BodyFunc == bf {
				return true
			}
		}
		return false
	}

	skills := NewSkills()

	// 腕の骨折は操作機能を下げる。近接命中は操作機能を畳むので操作の内訳が載る
	hsArm := &HealthStatus{}
	hsArm.Parts[BodyPartArms].SetCondition(HealthCondition{Type: ConditionType("fracture"), Severity: SeverityMedium})
	swordSrc := testSkillDefs.ProficiencySources(skills, nil, hsArm.BodyFuncs(testDefs), oapi.ProficiencyIdSwordAccuracy)
	assert.True(t, hasBodyFunc(swordSrc, BodyFuncManipulation), "近接は操作機能を畳む")

	// 頭の切り傷は視覚機能を下げる。遠隔命中は視覚機能を畳むので視覚の内訳が載る
	hsHead := &HealthStatus{}
	hsHead.Parts[BodyPartHead].SetCondition(HealthCondition{Type: ConditionType("laceration"), Timer: 60, Severity: TimerToSeverity(60)})
	bowSrc := testSkillDefs.ProficiencySources(skills, nil, hsHead.BodyFuncs(testDefs), oapi.ProficiencyIdBowAccuracy)
	assert.True(t, hasBodyFunc(bowSrc, BodyFuncSight), "遠隔は視覚機能を畳む")
}

func TestSkillDefsProficiencySources_目減りする機能を持たない倍率は身体機能を畳まない(t *testing.T) {
	t.Parallel()

	hs := &HealthStatus{}
	hs.Parts[BodyPartArms].SetCondition(HealthCondition{Type: ConditionType("fracture"), Severity: SeverityMedium})
	sources := testSkillDefs.ProficiencySources(NewSkills(), nil, hs.BodyFuncs(testDefs), oapi.ProficiencyIdSwordDamage)

	kinds := make([]ProficiencySourceKind, 0, len(sources))
	for _, src := range sources {
		kinds = append(kinds, src.Kind)
	}
	assert.Equal(t, []ProficiencySourceKind{SourceSkill}, kinds, "ダメージは腕の骨折でも操作機能を畳まない")
}

func TestSkillDefsProficiency_値は基準と内訳の和に一致する(t *testing.T) {
	t.Parallel()

	richSkills := NewSkills()
	richSkills.Get(oapi.SkillIdSword).Value = 3
	richSkills.Get(oapi.SkillIdBow).Value = 2
	richSkills.Get(oapi.SkillIdColdResist).Value = 4
	richSkills.Get(oapi.SkillIdNegotiation).Value = 5
	richSkills.Get(oapi.SkillIdStealth).Value = 6

	abils := &Abilities{
		Strength:  Ability{Total: 8},
		Sensation: Ability{Total: 6},
		Agility:   Ability{Total: 4},
		Dexterity: Ability{Total: 2},
		Vitality:  Ability{Total: 1},
	}

	sickHS := &HealthStatus{}
	sickHS.Parts[BodyPartWholeBody].SetCondition(HealthCondition{Type: ConditionType("hypothermia"), Severity: SeverityMedium})
	sickHS.Parts[BodyPartHead].SetCondition(HealthCondition{Type: ConditionType("laceration"), Timer: 60, Severity: TimerToSeverity(60)})

	cases := []struct {
		name      string
		skills    *Skills
		abils     *Abilities
		bodyFuncs BodyFuncs
	}{
		{"素の状態", NewSkills(), nil, HealthyBodyFuncs()},
		{"スキルと能力値", richSkills, abils, HealthyBodyFuncs()},
		{"不調あり", richSkills, abils, sickHS.BodyFuncs(testDefs)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// 表示は 基準 + Σ内訳、適用は CalcProficiencyValue。両者の一致が表示=適用の土台
			for key := range testSkillDefs.Proficiencies {
				sum := int(consts.PercentBase)
				for _, s := range testSkillDefs.ProficiencySources(tc.skills, tc.abils, tc.bodyFuncs, key) {
					sum += s.Value
				}
				assert.Equal(t, int(testSkillDefs.ProficiencyValue(tc.skills, tc.abils, tc.bodyFuncs, key)), sum,
					"キー %s で 最終値 = 基準 + Σ内訳 が破れた", key)
			}
		})
	}
}

func TestSkillDefsProficiencyValue_ElementResistAllTypes(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(oapi.SkillIdFireResist).Value = 2
	skills.Get(oapi.SkillIdThunderResist).Value = 4
	skills.Get(oapi.SkillIdChillResist).Value = 6
	skills.Get(oapi.SkillIdPhotonResist).Value = 8

	// 各元素耐性: 100 + Lv*(-3)
	assert.Equal(t, 94, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdFireResist)))    // 100 + 2*(-3)
	assert.Equal(t, 88, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdThunderResist))) // 100 + 4*(-3)
	assert.Equal(t, 82, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdChillResist)))   // 100 + 6*(-3)
	assert.Equal(t, 76, int(testSkillDefs.ProficiencyValue(skills, nil, HealthyBodyFuncs(), oapi.ProficiencyIdPhotonResist)))  // 100 + 8*(-3)
}
