package raw

import (
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeDefRaws は validateCodeRows と validateConditions を通る最小の raws を作り、mutate で崩す
func makeDefRaws(mutate func(*rawschema.Raws)) rawschema.Raws {
	bodyParts := make([]rawschema.BodyPart, 0, gc.BodyPartCount)
	for bp := range gc.BodyPartCount {
		bodyParts = append(bodyParts, rawschema.BodyPart{Id: bp.ID(), Name: bp.ID(), HitWeight: 1})
	}
	attackType := func(id rawschema.AttackCategory) rawschema.AttackType {
		return rawschema.AttackType{
			Id:       id,
			Name:     string(id),
			Range:    rawschema.AttackRangeMelee,
			Injury:   "fracture",
			Skill:    string(id),
			Damage:   rawschema.ProficiencyId(id + "_damage"),
			Accuracy: rawschema.ProficiencyId(id + "_accuracy"),
		}
	}
	skill := func(id rawschema.SkillId) rawschema.Skill {
		return rawschema.Skill{Id: id, Name: id, Ability: rawschema.Str, Category: "general", Proficiencies: &[]rawschema.SkillProficiency{
			{Id: rawschema.ProficiencyId(id + "_damage"), PerLevel: 1},
			{Id: rawschema.ProficiencyId(id + "_accuracy"), PerLevel: 1},
		}}
	}
	raws := rawschema.Raws{
		BodyAttacks: &[]rawschema.BodyAttack{{Id: BareHands, Name: "Bare hands"}},
		Dungeons:    &[]rawschema.Dungeon{{Id: "dead_forest"}},
		Settlements: &[]rawschema.Settlement{{Id: "dwelling", Weight: 1}},
		BodyParts:   &bodyParts,
		Conditions: &[]rawschema.Condition{
			{Id: "hypothermia", Name: "Hypothermia", Description: "Hypothermia"},
			{Id: "fracture", Name: "Fracture", Description: "Fracture", Parts: &[]rawschema.BodyPartId{"arm"}},
		},
		AttackTypes: &[]rawschema.AttackType{attackType(rawschema.AttackCategorySword), attackType(rawschema.AttackCategorySpear)},
		Skills:      &[]rawschema.Skill{skill("sword"), skill("spear")},
		Members:     &[]rawschema.Member{{Id: "ash", Player: new(true)}},
	}
	mutate(&raws)
	return raws
}

func TestValidateConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rawschema.Raws)
		wantErr error
	}{
		{
			name:   "全行がそろえば通る",
			mutate: func(*rawschema.Raws) {},
		},
		{
			name: "部位を限った不調は命中しうる部位があれば通る",
			mutate: func(r *rawschema.Raws) {
				(*r.Conditions)[0].Parts = &[]rawschema.BodyPartId{"arm", "leg"}
			},
		},
		{
			name: "Go に無い部位の行",
			mutate: func(r *rawschema.Raws) {
				*r.BodyParts = append(*r.BodyParts, rawschema.BodyPart{Id: "tail", Name: "Tail"})
			},
			wantErr: errBodyPartUnknown,
		},
		{
			name: "不調の部位に全身を含む",
			mutate: func(r *rawschema.Raws) {
				(*r.Conditions)[0].Parts = &[]rawschema.BodyPartId{"arm", "whole_body"}
			},
			wantErr: errConditionPartWholeBody,
		},
		{
			name: "不調の部位がどれも命中しない",
			mutate: func(r *rawschema.Raws) {
				(*r.BodyParts)[gc.BodyPartArms].HitWeight = 0
				(*r.Conditions)[0].Parts = &[]rawschema.BodyPartId{"arm"}
			},
			wantErr: errConditionNoHitPart,
		},
		{
			name: "怪我が部位を持たない",
			mutate: func(r *rawschema.Raws) {
				(*r.Conditions)[1].Parts = nil
			},
			wantErr: errInjuryWithoutParts,
		},
		{
			name: "部位を持たない全身性の不調は命中の重みによらず通る",
			mutate: func(r *rawschema.Raws) {
				for i := range *r.BodyParts {
					(*r.BodyParts)[i].HitWeight = 0
				}
				(*r.Conditions)[1].Parts = &[]rawschema.BodyPartId{"arm"}
				(*r.BodyParts)[gc.BodyPartArms].HitWeight = 1
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateConditions(makeDefRaws(tt.mutate))
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateCodeRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rawschema.Raws)
		wantErr error
	}{
		{
			name:   "Go の不調と部位すべてに行があれば通る",
			mutate: func(*rawschema.Raws) {},
		},
		{
			name: "プレイヤーの member が無い",
			mutate: func(r *rawschema.Raws) {
				(*r.Members)[0].Player = nil
			},
			wantErr: errPlayerMemberMissing,
		},
		{
			name: "プレイヤーの member が2つある",
			mutate: func(r *rawschema.Raws) {
				*r.Members = append(*r.Members, rawschema.Member{Id: "ash2", Player: new(true)})
			},
			wantErr: errPlayerMemberDuplicate,
		},
		{
			name: "部位の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.BodyParts = (*r.BodyParts)[:gc.BodyPartWholeBody]
			},
			wantErr: errBodyPartRowMissing,
		},
		{
			name: "近接の攻撃種別が射程を持つ",
			mutate: func(r *rawschema.Raws) {
				(*r.AttackTypes)[0].MaxRange = new(5)
			},
			wantErr: errAttackRangeParams,
		},
		{
			name: "遠距離の攻撃種別が射程を持たない",
			mutate: func(r *rawschema.Raws) {
				(*r.AttackTypes)[0].Range = rawschema.AttackRangeRanged
			},
			wantErr: errAttackRangeParams,
		},
		{
			name: "武器の効果倍率を別のスキルが持つ",
			mutate: func(r *rawschema.Raws) {
				skills := *r.Skills
				sword := slices.IndexFunc(skills, func(s rawschema.Skill) bool { return s.Id == "sword" })
				spear := slices.IndexFunc(skills, func(s rawschema.Skill) bool { return s.Id == "spear" })
				skills[sword].Proficiencies, skills[spear].Proficiencies = skills[spear].Proficiencies, skills[sword].Proficiencies
			},
			wantErr: errWeaponProficiencyOwner,
		},
		{
			name: "集落が無い",
			mutate: func(r *rawschema.Raws) {
				r.Settlements = &[]rawschema.Settlement{}
			},
			wantErr: errTableMissing,
		},
		{
			name: "ダンジョンが無い",
			mutate: func(r *rawschema.Raws) {
				r.Dungeons = nil
			},
			wantErr: errTableMissing,
		},
		{
			name: "素手の体の攻撃が無い",
			mutate: func(r *rawschema.Raws) {
				r.BodyAttacks = nil
			},
			wantErr: errBareHandsMissing,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateCodeRows(makeDefRaws(tt.mutate))
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewSkillDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	defs, err := NewSkillDefs(raws)
	require.NoError(t, err)

	sword, ok := defs.Skills["sword"]
	require.True(t, ok)
	assert.Equal(t, gc.SkillDef{
		Name:     "Swordsmanship",
		Summary:  "Technique for handling swords",
		GainedBy: "Raised by attacking with swords",
		Effect:   "Increases sword damage and accuracy",
		Ability:  rawschema.Str,
	}, sword)

	assert.Equal(t, []gc.SkillCategory{
		{Name: "Melee", IDs: []rawschema.SkillId{"sword", "spear", "fist"}},
		{Name: "Ranged", IDs: []rawschema.SkillId{"bow", "handgun", "rifle", "cannon", "throw"}},
		{Name: "Craft", IDs: []rawschema.SkillId{"crafting", "smithing", "negotiation", "mechanic"}},
		{Name: "Mobility", IDs: []rawschema.SkillId{"sprinting", "stealth", "night_vision", "weight_bearing"}},
		{Name: "Survival", IDs: []rawschema.SkillId{"cold_resist", "hunger_resist", "healing", "exploration"}},
		{Name: "Defense", IDs: []rawschema.SkillId{"heavy_armor", "fire_resist", "thunder_resist", "chill_resist", "photon_resist"}},
	}, defs.Categories)

	weapons := map[string][3]string{}
	accuracyLimbs := map[string]gc.BodyFuncKind{}
	for _, w := range defs.Weapons {
		weapons[string(w.AttackType)] = [3]string{w.Skill, string(w.Damage), string(w.Accuracy)}
		accuracyLimbs[string(w.AttackType)] = defs.Proficiencies[w.Accuracy].BodyFunc
	}
	assert.Equal(t, map[string][3]string{
		"sword":   {"sword", "sword_damage", "sword_accuracy"},
		"spear":   {"spear", "spear_damage", "spear_accuracy"},
		"fist":    {"fist", "fist_damage", "fist_accuracy"},
		"bow":     {"bow", "bow_damage", "bow_accuracy"},
		"handgun": {"handgun", "handgun_damage", "handgun_accuracy"},
		"rifle":   {"rifle", "rifle_damage", "rifle_accuracy"},
		"cannon":  {"cannon", "cannon_damage", "cannon_accuracy"},
		"throw":   {"throw", "throw_damage", "throw_accuracy"},
	}, weapons)
	assert.Equal(t, map[string]gc.BodyFuncKind{
		"sword": gc.BodyFuncManipulation, "spear": gc.BodyFuncManipulation, "fist": gc.BodyFuncManipulation,
		"bow": gc.BodyFuncSight, "handgun": gc.BodyFuncSight, "rifle": gc.BodyFuncSight, "cannon": gc.BodyFuncSight, "throw": gc.BodyFuncSight,
	}, accuracyLimbs, "命中は近接なら操作、遠隔なら視覚で目減りする")
	assert.Equal(t, gc.BodyFuncKind(""), defs.Proficiencies[rawschema.ProficiencyIdSwordDamage].BodyFunc, "ダメージは身体機能で目減りしない")
	assert.Equal(t, []gc.ResistDef{
		{Element: rawschema.ElementFIRE, Proficiency: rawschema.ProficiencyIdFireResist},
		{Element: rawschema.ElementTHUNDER, Proficiency: rawschema.ProficiencyIdThunderResist},
		{Element: rawschema.ElementCHILL, Proficiency: rawschema.ProficiencyIdChillResist},
		{Element: rawschema.ElementPHOTON, Proficiency: rawschema.ProficiencyIdPhotonResist},
	}, defs.Resists)

	skills := gc.NewSkills()
	for _, s := range PtrSlice(raws.Skills) {
		skills.Get(s.Id).Value = 10
	}
	got := map[rawschema.ProficiencyId]int{}
	for _, key := range rawschema.ProficiencyIdValues() {
		got[key] = int(defs.ProficiencyValue(skills, nil, gc.HealthyBodyFuncs(), key))
	}
	assert.Equal(t, map[rawschema.ProficiencyId]int{
		rawschema.ProficiencyIdFireResist: 70, rawschema.ProficiencyIdThunderResist: 70, rawschema.ProficiencyIdChillResist: 70, rawschema.ProficiencyIdPhotonResist: 70,
		rawschema.ProficiencyIdColdProgress: 70, rawschema.ProficiencyIdHungerProgress: 80, rawschema.ProficiencyIdHealingEffect: 150, rawschema.ProficiencyIdMaxWeight: 140,
		rawschema.ProficiencyIdExploration: 140, rawschema.ProficiencyIdEnemyVision: 70, rawschema.ProficiencyIdNightVision: 150, rawschema.ProficiencyIdMoveCost: 80,
		rawschema.ProficiencyIdCraftCost: 70, rawschema.ProficiencyIdSmithQuality: 130, rawschema.ProficiencyIdBuyPrice: 80, rawschema.ProficiencyIdSellPrice: 120, rawschema.ProficiencyIdHeavyArmor: 50,
		rawschema.ProficiencyIdSwordDamage: 150, rawschema.ProficiencyIdSpearDamage: 150, rawschema.ProficiencyIdFistDamage: 150, rawschema.ProficiencyIdBowDamage: 150,
		rawschema.ProficiencyIdHandgunDamage: 150, rawschema.ProficiencyIdRifleDamage: 150, rawschema.ProficiencyIdCannonDamage: 150, rawschema.ProficiencyIdThrowDamage: 150,
		rawschema.ProficiencyIdSwordAccuracy: 130, rawschema.ProficiencyIdSpearAccuracy: 130, rawschema.ProficiencyIdFistAccuracy: 130, rawschema.ProficiencyIdBowAccuracy: 130,
		rawschema.ProficiencyIdHandgunAccuracy: 130, rawschema.ProficiencyIdRifleAccuracy: 130, rawschema.ProficiencyIdCannonAccuracy: 130, rawschema.ProficiencyIdThrowAccuracy: 130,
	}, got)
}

func TestNewConditionDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	defs, err := NewConditionDefs(raws)
	require.NoError(t, err)

	tests := []struct {
		ct   gc.ConditionType
		want gc.ConditionDef
	}{
		{gc.ConditionType("hypothermia"), gc.ConditionDef{
			Name: "Hypothermia", Description: "The body is dangerously cold. Warm up to recover.",
			PainPerSeverity: 6, BodyFuncDropPerSeverity: 10, BloodDropPerSeverity: 22, Cause: gc.CauseHypothermia, Onset: rawschema.Cold,
		}},
		{gc.ConditionType("fracture"), gc.ConditionDef{
			Name: "Fracture", Description: "A broken bone. It will not heal until treated.",
			PainPerSeverity: 18, BodyFuncDropPerSeverity: 20, Recovery: rawschema.RecoverAfterTend, RecoverPer: 1,
			Parts: []gc.BodyPart{gc.BodyPartArms, gc.BodyPartHands, gc.BodyPartLegs, gc.BodyPartFeet},
		}},
		{gc.ConditionType("laceration"), gc.ConditionDef{
			Name: "Laceration", Description: "An open wound. It will not heal until treated.",
			PainPerSeverity: 8, BodyFuncDropPerSeverity: 8, Recovery: rawschema.RecoverAfterTend, RecoverPer: 2,
			BloodDropPerSeverity: 25, Cause: gc.CauseBloodLoss,
			Parts: []gc.BodyPart{gc.BodyPartTorso, gc.BodyPartHead, gc.BodyPartArms, gc.BodyPartHands, gc.BodyPartLegs, gc.BodyPartFeet},
		}},
		{gc.ConditionType("liver_illness"), gc.ConditionDef{
			Name: "Liver illness", Description: "It worsens while untreated and drains HP when severe.",
			PainPerSeverity: 4, BodyFuncDropPerSeverity: 10, Recovery: rawschema.ProgressUntilTend, WorsenPer: 2, RecoverPer: 1,
			BloodDropPerSeverity: 25, Cause: gc.CauseIllness,
		}},
		{gc.ConditionType("food_poisoning"), gc.ConditionDef{
			Name: "Food poisoning", Description: "Nausea from bad food. It clears on its own over time.",
			PainPerSeverity: 5, BodyFuncDropPerSeverity: 12, Recovery: rawschema.RecoverOverTime, RecoverPer: 2, Onset: rawschema.RottenFood,
		}},
		{gc.ConditionType("exhaustion"), gc.ConditionDef{
			Name: "Exhaustion", Description: "Worn out from lack of rest. Sleep to recover.",
			BodyFuncDropPerSeverity: 10, Onset: rawschema.Fatigue,
		}},
		{gc.ConditionType("malnutrition"), gc.ConditionDef{
			Name: "Malnutrition", Description: "Weakened by hunger. Eat to recover.",
			BodyFuncDropPerSeverity: 10, Onset: rawschema.Hunger,
		}},
	}
	for _, tt := range tests {
		t.Run(string(tt.ct), func(t *testing.T) {
			t.Parallel()
			got, ok := defs.Def(tt.ct)
			require.True(t, ok)
			assert.Equal(t, tt.want, got)
		})
	}

	var weights [gc.BodyPartCount]float64
	var names [gc.BodyPartCount]string
	for bp := range gc.BodyPartCount {
		weights[bp] = defs.HitWeight(bp)
		names[bp] = string(defs.PartName(bp))
	}
	assert.Equal(t, [gc.BodyPartCount]float64{2, 4, 3, 1, 3, 1, 0}, weights)
	assert.Equal(t, [gc.BodyPartCount]string{"Head", "Torso", "Arm", "Hand", "Leg", "Foot", "Whole body"}, names)
}

// mapKeys は文字列のキーを持つ map のキーを返す
func mapKeys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestOrdinalEnumMaps_スキーマのenumの全値を持つ(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, rawschema.WeatherIdValues(), mapKeys(weatherKinds))
	assert.ElementsMatch(t, rawschema.SeasonIdValues(), mapKeys(seasons))
	assert.ElementsMatch(t, rawschema.TimeOfDayIdValues(), mapKeys(timesOfDay))
	assert.ElementsMatch(t, rawschema.ConditionSeverityValues(), mapKeys(severities))
}

func TestConditionDeathCauseValues_不調の死因はどれも死因の定数に当たる(t *testing.T) {
	t.Parallel()

	values := rawschema.ConditionDeathCauseValues()
	got := make([]gc.DeathCause, 0, len(values))
	for _, v := range values {
		got = append(got, gc.DeathCause(v))
	}
	assert.Equal(t, []gc.DeathCause{gc.CauseHypothermia, gc.CauseIllness, gc.CauseBloodLoss}, got,
		"raw の値をそのまま死因にするので、列挙値と定数の文字列が一致しないと表示名が引けない")
}
