package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcCharModifiers_AllSkillsZero(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	mods := CalcCharModifiers(skills, nil, nil)

	// スキル値0のとき全倍率は100（等倍）
	for _, id := range WeaponSkillIDs {
		assert.Equal(t, 100, int(mods.Value(WeaponDamageKey(id))), "武器ダメージ %s は100", id)
		assert.Equal(t, 100, int(mods.Value(WeaponAccuracyKey(id))), "武器命中 %s は100", id)
	}
	assert.Equal(t, 100, int(mods.Value(ModColdProgress)))
	assert.Equal(t, 100, int(mods.Value(ModHungerProgress)))
	assert.Equal(t, 100, int(mods.Value(ModHealingEffect)))
	assert.Equal(t, 100, int(mods.Value(ModMaxWeight)))
	assert.Equal(t, 100, int(mods.Value(ModEnemyVision)))
	assert.Equal(t, 100, int(mods.Value(ModMoveCost)))
	assert.Equal(t, 100, int(mods.Value(ModCraftCost)))
	assert.Equal(t, 100, int(mods.Value(ModSmithQuality)))
	assert.Equal(t, 100, int(mods.Value(ModBuyPrice)))
	assert.Equal(t, 100, int(mods.Value(ModSellPrice)))
	assert.Equal(t, 100, int(mods.Value(ModHeavyArmor)))
}

func TestCalcCharModifiers_SkillEffects(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 2

	mods := CalcCharModifiers(skills, nil, nil)

	// 刀剣Lv2: ダメージ倍率 = 100 + 2*5 = 110
	assert.Equal(t, 110, int(mods.Value(ModSwordDamage)))
	// 刀剣Lv2: 命中倍率 = 100 + 2*3 = 106
	assert.Equal(t, 106, int(mods.Value(ModSwordAccuracy)))
	// 他の武器は影響なし
	assert.Equal(t, 100, int(mods.Value(ModSpearDamage)))
}

func TestCalcCharModifiers_NegativeCoefficient(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillColdResist).Value = 3

	mods := CalcCharModifiers(skills, nil, nil)

	// 耐寒Lv3: 低体温進行 = 100 + 3*(-3) = 91
	assert.Equal(t, 91, int(mods.Value(ModColdProgress)))
	// 耐寒Lv3: 火耐性 = 100 + 0*(-3) = 100（SkillFireResistはLv0のまま）
	assert.Equal(t, 100, int(mods.Value(ModFireResist)))
}

func TestCalcCharModifiers_WithAbilities(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 2

	abils := &Abilities{
		Strength: Ability{Total: 10},
	}

	mods := CalcCharModifiers(skills, abils, nil)

	// 刀剣Lv2 + STR10: ダメージ = 100 + 2*5 + 10*1 = 120
	assert.Equal(t, 120, int(mods.Value(ModSwordDamage)))
	// 刀剣Lv2 + STR10: 命中 = 100 + 2*3 + 10*1 = 116
	assert.Equal(t, 116, int(mods.Value(ModSwordAccuracy)))
}

func TestCalcCharModifiers_AbilityNegativeDirection(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillColdResist).Value = 1

	abils := &Abilities{
		Vitality: Ability{Total: 5},
	}

	mods := CalcCharModifiers(skills, abils, nil)

	// 耐寒Lv1 + VIT5: 低体温進行 = 100 + 1*(-3) + 5*(-1) = 92
	assert.Equal(t, 92, int(mods.Value(ModColdProgress)))
}

func TestCalcCharModifiers_Sources(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillSword).Value = 3

	abils := &Abilities{
		Strength: Ability{Total: 8},
	}

	mods := CalcCharModifiers(skills, abils, nil)

	sources := mods.Sources[ModSwordDamage]
	assert.Len(t, sources, 2, "スキルと能力値の2つのソースがある")
	assert.Equal(t, "Swordsmanship Lv3", sources[0].Label)
	assert.Equal(t, 15, sources[0].Value) // 3*5
	assert.Equal(t, "STR 8", sources[1].Label)
	assert.Equal(t, 8, sources[1].Value) // 8*1
}

func TestCalcCharModifiers_HealthPenalty(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeverityMedium,
	})

	mods := CalcCharModifiers(skills, nil, hs)

	// 不調は MoveCost へ直接足さず身体機能 Capacities に一本化する
	assert.Equal(t, 100, int(mods.Value(ModMoveCost)), "低体温は MoveCost へ足さない")
	// 中度の全身性低体温 6/10: 痛み6*2=12、意識=100-20-12/2=74。
	// 局所低下は無いので操作・歩行・視覚はいずれも意識乗数だけを受けて74
	assert.Equal(t, BodyCapacities{Pain: 12, Blood: 100, Consciousness: 74, Manipulation: 74, Moving: 74, Sight: 74}, mods.Capacities)
}

func TestCalcCharModifiers_NilAbilsAndHS(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	// panicしないことを確認
	mods := CalcCharModifiers(skills, nil, nil)
	assert.NotNil(t, mods)
}

func TestCalcCharModifiers_Negotiation(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillNegotiation).Value = 4

	mods := CalcCharModifiers(skills, nil, nil)

	// 交渉Lv4: 買値 = 100 + 4*(-2) = 92 (安く買える)
	assert.Equal(t, 92, int(mods.Value(ModBuyPrice)))
	// 交渉Lv4: 売値 = 100 + 4*2 = 108 (高く売れる)
	assert.Equal(t, 108, int(mods.Value(ModSellPrice)))
}

func TestCalcCharModifiers_MultipleSkills(t *testing.T) {
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

	mods := CalcCharModifiers(skills, abils, nil)

	// 刀剣Lv3 + STR8: ダメージ = 100 + 3*5 + 8*1 = 123
	assert.Equal(t, 123, int(mods.Value(ModSwordDamage)))
	// 拳銃Lv5 + SEN6: ダメージ = 100 + 5*5 + 6*1 = 131
	assert.Equal(t, 131, int(mods.Value(ModHandgunDamage)))
	// クラフトLv2 + DEX4: 素材消費 = 100 + 2*(-3) + 4*(-1) = 90
	assert.Equal(t, 90, int(mods.Value(ModCraftCost)))
	// 長物は未使用: ダメージ = 100 + 0*5 + 8*1 = 108（STR能力値のみ）
	assert.Equal(t, 108, int(mods.Value(ModSpearDamage)))
}

func TestCalcCharModifiers_AllFactors(t *testing.T) {
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

	mods := CalcCharModifiers(skills, abils, hs)

	// 走破Lv4 + AGI10: MoveCost = 100 + 4*(-2) + 10*(-1) = 82。低体温は MoveCost へ足さない
	assert.Equal(t, 82, int(mods.Value(ModMoveCost)))
	// 重度の全身性低体温 6/10 は身体機能へ効く。意識=100-30-18/2=61、歩行=100*61/100=61
	assert.Equal(t, 61, int(mods.Capacities.Moving))

	// Sourcesはスキルと能力値の2要因。健康は Capacities 側なので MoveCost には載らない
	sources := mods.Sources[ModMoveCost]
	assert.Len(t, sources, 2, "スキルと能力値の2つのソース")
}

func TestCalcCharModifiers_FireAbility(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillRifle).Value = 4

	abils := &Abilities{
		Sensation: Ability{Total: 12},
	}

	mods := CalcCharModifiers(skills, abils, nil)

	// 小銃Lv4 + SEN12: ダメージ = 100 + 4*5 + 12*1 = 132
	assert.Equal(t, 132, int(mods.Value(ModRifleDamage)))
	// 小銃Lv4 + SEN12: 命中 = 100 + 4*3 + 12*1 = 124
	assert.Equal(t, 124, int(mods.Value(ModRifleAccuracy)))
}

func TestCalcCharModifiers_AccuracyFoldsCapacity(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	hs := &HealthStatus{
		Parts: [BodyPartCount]BodyPartHealth{},
	}
	hs.Parts[BodyPartWholeBody].SetCondition(HealthCondition{
		Type:     ConditionHypothermia,
		Severity: SeverityMedium,
	})

	mods := CalcCharModifiers(skills, nil, hs)

	// 中度の全身性低体温で操作・視覚は74。スキルLv0の基礎命中100×74%=74
	assert.Equal(t, 74, int(mods.Value(ModSwordAccuracy)), "近接は操作機能を畳み込む")
	assert.Equal(t, 74, int(mods.Value(ModBowAccuracy)), "遠隔は視覚機能を畳み込む")

	// 内訳の末尾に身体機能の加法差分が載る。100→74 なので -26
	swordSrc := mods.Sources[ModSwordAccuracy]
	assert.Equal(t, "Manipulation 74%", swordSrc[len(swordSrc)-1].Label)
	assert.Equal(t, -26, swordSrc[len(swordSrc)-1].Value)
	bowSrc := mods.Sources[ModBowAccuracy]
	assert.Equal(t, "Sight 74%", bowSrc[len(bowSrc)-1].Label)
	assert.Equal(t, -26, bowSrc[len(bowSrc)-1].Value)
}

func TestCalcCharModifiers_ElementResistAllTypes(t *testing.T) {
	t.Parallel()

	skills := NewSkills()
	skills.Get(SkillFireResist).Value = 2
	skills.Get(SkillThunderResist).Value = 4
	skills.Get(SkillChillResist).Value = 6
	skills.Get(SkillPhotonResist).Value = 8

	mods := CalcCharModifiers(skills, nil, nil)

	// 各元素耐性: 100 + Lv*(-3)
	assert.Equal(t, 94, int(mods.Value(ModFireResist)))    // 100 + 2*(-3)
	assert.Equal(t, 88, int(mods.Value(ModThunderResist))) // 100 + 4*(-3)
	assert.Equal(t, 82, int(mods.Value(ModChillResist)))   // 100 + 6*(-3)
	assert.Equal(t, 76, int(mods.Value(ModPhotonResist)))  // 100 + 8*(-3)
}
