package raw

import (
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeDefRaws は不調を起こす仕組みすべてと、Go の部位、攻撃種別、スキル、熟練の効果倍率すべてに行を持つ検証を通る raws を作り、mutate で崩す
func makeDefRaws(mutate func(*oapi.Raws)) oapi.Raws {
	bodyParts := make([]oapi.BodyPart, 0, gc.BodyPartCount)
	for bp := range gc.BodyPartCount {
		bodyParts = append(bodyParts, oapi.BodyPart{Id: bp.ID(), Name: bp.ID(), HitWeight: 1})
	}
	onset := func(o oapi.ConditionOnset) *oapi.ConditionOnset { return &o }
	conditions := []oapi.Condition{
		{Id: "hypothermia", Name: "Hypothermia", Description: "Hypothermia", Onset: onset(oapi.Cold)},
		{Id: "fracture", Name: "Fracture", Description: "Fracture"},
		{Id: "food_poisoning", Name: "Food poisoning", Description: "Food poisoning", Onset: onset(oapi.RottenFood)},
		{Id: "exhaustion", Name: "Exhaustion", Description: "Exhaustion", Onset: onset(oapi.Fatigue)},
		{Id: "malnutrition", Name: "Malnutrition", Description: "Malnutrition", Onset: onset(oapi.Hunger)},
	}
	attackTypes := make([]oapi.AttackType, 0, len(gc.AllAttackTypes))
	for _, at := range gc.AllAttackTypes {
		attackTypes = append(attackTypes, oapi.AttackType{Id: oapi.AttackCategory(at.Type), Injury: string(gc.ConditionType("fracture"))})
	}
	skills := make([]oapi.Skill, 0, len(gc.AllSkillIDs))
	for _, id := range gc.AllSkillIDs {
		skills = append(skills, oapi.Skill{Id: string(id), Name: string(id), Ability: oapi.Str, Category: "general", Proficiencies: &[]oapi.SkillProficiency{}})
	}
	for _, key := range gc.ProficiencyKeys() {
		owner := 0
		for i, id := range gc.WeaponSkillIDs {
			if key == gc.WeaponDamageKey(id) || key == gc.WeaponAccuracyKey(id) {
				owner = slices.Index(gc.AllSkillIDs, gc.WeaponSkillIDs[i])
			}
		}
		*skills[owner].Proficiencies = append(*skills[owner].Proficiencies, oapi.SkillProficiency{Id: string(key), PerLevel: 1})
	}
	raws := oapi.Raws{
		BodyParts:       &bodyParts,
		Conditions:      &conditions,
		AttackTypes:     &attackTypes,
		Skills:          &skills,
		SkillCategories: &[]oapi.SkillCategory{{Id: "general", Name: "General"}},
		Items: &[]oapi.Item{
			{Id: "splint", Remedy: &oapi.Remedy{Treats: []oapi.ConditionTypeName{string(gc.ConditionType("fracture"))}}},
			{Id: "manual", Book: &oapi.Book{Skill: &oapi.SkillBook{TargetSkill: string(gc.SkillSword)}}},
		},
		Professions:     &[]oapi.Profession{{Id: "soldier", Skills: &[]oapi.ProfessionSkill{{Id: string(gc.SkillRifle)}}}},
		InteriorProfile: &oapi.InteriorProfile{},
		Weathers:        &[]oapi.Weather{{Id: oapi.Clear}},
		HungerStages:    &[]oapi.HungerStage{{Id: oapi.HungerStageIdStarving}},
		FatigueStages:   &[]oapi.FatigueStage{{Id: oapi.FatigueStageIdRested}},
	}
	mutate(&raws)
	return raws
}

func TestValidateConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*oapi.Raws)
		wantErr error
	}{
		{
			name:   "全行がそろい参照が実在すれば通る",
			mutate: func(*oapi.Raws) {},
		},
		{
			name: "部位を限った不調は命中しうる部位があれば通る",
			mutate: func(r *oapi.Raws) {
				(*r.Conditions)[0].Parts = &[]oapi.BodyPartId{"arm", "leg"}
			},
		},
		{
			name: "部位の id 重複",
			mutate: func(r *oapi.Raws) {
				*r.BodyParts = append(*r.BodyParts, oapi.BodyPart{Id: "arm", Name: "Arm"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "不調の id 重複",
			mutate: func(r *oapi.Raws) {
				*r.Conditions = append(*r.Conditions, oapi.Condition{Id: "fracture"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "Go に無い部位の行",
			mutate: func(r *oapi.Raws) {
				*r.BodyParts = append(*r.BodyParts, oapi.BodyPart{Id: "tail", Name: "Tail"})
			},
			wantErr: errBodyPartUnknown,
		},
		{
			name: "不調の部位が実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.Conditions)[0].Parts = &[]oapi.BodyPartId{"tail"}
			},
			wantErr: errConditionPartUndefined,
		},
		{
			name: "不調の部位に全身を含む",
			mutate: func(r *oapi.Raws) {
				(*r.Conditions)[0].Parts = &[]oapi.BodyPartId{"arm", "whole_body"}
			},
			wantErr: errConditionPartWholeBody,
		},
		{
			name: "不調の部位がどれも命中しない",
			mutate: func(r *oapi.Raws) {
				(*r.BodyParts)[gc.BodyPartArms].HitWeight = 0
				(*r.Conditions)[0].Parts = &[]oapi.BodyPartId{"arm"}
			},
			wantErr: errConditionNoHitPart,
		},
		{
			name: "全部位の不調でも命中する部位が無い",
			mutate: func(r *oapi.Raws) {
				for i := range *r.BodyParts {
					(*r.BodyParts)[i].HitWeight = 0
				}
			},
			wantErr: errConditionNoHitPart,
		},
		{
			name: "攻撃種別の怪我が実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.AttackTypes)[0].Injury = "bruise"
			},
			wantErr: errAttackTypeInjuryUndefined,
		},
		{
			name: "攻撃種別の id 重複",
			mutate: func(r *oapi.Raws) {
				*r.AttackTypes = append(*r.AttackTypes, (*r.AttackTypes)[0])
			},
			wantErr: errDuplicateID,
		},
		{
			name: "治療の対象が実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.Items)[0].Remedy.Treats = []oapi.ConditionTypeName{"Fracture"}
			},
			wantErr: errRemedyConditionUndefined,
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
		mutate  func(*oapi.Raws)
		wantErr error
	}{
		{
			name:   "Go の不調と部位すべてに行があれば通る",
			mutate: func(*oapi.Raws) {},
		},
		{
			name: "部位の行が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.BodyParts = (*r.BodyParts)[:gc.BodyPartWholeBody]
			},
			wantErr: errBodyPartRowMissing,
		},
		{
			name: "仕組みを持つ不調が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.Conditions = (*r.Conditions)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "同じ仕組みを2つの不調が持つ",
			mutate: func(r *oapi.Raws) {
				(*r.Conditions)[1].Onset = (*r.Conditions)[0].Onset
			},
			wantErr: errDuplicateID,
		},
		{
			name: "攻撃種別の行が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.AttackTypes = (*r.AttackTypes)[1:]
			},
			wantErr: errAttackTypeRowMissing,
		},
		{
			name: "スキルの行が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.Skills = (*r.Skills)[:len(*r.Skills)-1]
			},
			wantErr: errSkillRowMissing,
		},
		{
			name: "Go に無いスキルの行",
			mutate: func(r *oapi.Raws) {
				*r.Skills = append(*r.Skills, oapi.Skill{Id: "juggling", Ability: oapi.Dex, Category: "general"})
			},
			wantErr: errSkillUnknown,
		},
		{
			name: "熟練の効果倍率の行が欠ける",
			mutate: func(r *oapi.Raws) {
				*(*r.Skills)[0].Proficiencies = (*(*r.Skills)[0].Proficiencies)[1:]
			},
			wantErr: errProficiencyRowMissing,
		},
		{
			name: "Go に無い熟練の効果倍率の行",
			mutate: func(r *oapi.Raws) {
				*(*r.Skills)[0].Proficiencies = append(*(*r.Skills)[0].Proficiencies, oapi.SkillProficiency{Id: "luck"})
			},
			wantErr: errProficiencyUnknown,
		},
		{
			name: "武器の効果倍率を別のスキルが持つ",
			mutate: func(r *oapi.Raws) {
				sword := slices.Index(gc.AllSkillIDs, gc.SkillSword)
				spear := slices.Index(gc.AllSkillIDs, gc.SkillSpear)
				(*r.Skills)[sword].Proficiencies, (*r.Skills)[spear].Proficiencies = (*r.Skills)[spear].Proficiencies, (*r.Skills)[sword].Proficiencies
			},
			wantErr: errWeaponProficiencyOwner,
		},
		{
			name: "生活感の表が無い",
			mutate: func(r *oapi.Raws) {
				r.InteriorProfile = nil
			},
			wantErr: errTableMissing,
		},
		{
			name: "天候の表が空",
			mutate: func(r *oapi.Raws) {
				r.Weathers = &[]oapi.Weather{}
			},
			wantErr: errTableMissing,
		},
		{
			name: "疲労の段階が無い",
			mutate: func(r *oapi.Raws) {
				r.FatigueStages = nil
			},
			wantErr: errTableMissing,
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

func TestValidateSkills(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*oapi.Raws)
		wantErr error
	}{
		{
			name:   "スキルと参照が実在すれば通る",
			mutate: func(*oapi.Raws) {},
		},
		{
			name: "スキルの id 重複",
			mutate: func(r *oapi.Raws) {
				*r.Skills = append(*r.Skills, oapi.Skill{Id: "sword", Category: "general"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "カテゴリの id 重複",
			mutate: func(r *oapi.Raws) {
				*r.SkillCategories = append(*r.SkillCategories, oapi.SkillCategory{Id: "general"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "熟練の効果倍率の id がスキルをまたいで重複",
			mutate: func(r *oapi.Raws) {
				*(*r.Skills)[1].Proficiencies = append(*(*r.Skills)[1].Proficiencies, (*(*r.Skills)[0].Proficiencies)[0])
			},
			wantErr: errDuplicateID,
		},
		{
			name: "スキルのカテゴリが実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.Skills)[0].Category = "magic"
			},
			wantErr: errSkillCategoryUndefined,
		},
		{
			name: "本のスキルが実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.Items)[1].Book.Skill.TargetSkill = "juggling"
			},
			wantErr: errBookSkillUndefined,
		},
		{
			name: "職業のスキルが実在しない",
			mutate: func(r *oapi.Raws) {
				(*(*r.Professions)[0].Skills)[0].Id = "juggling"
			},
			wantErr: errProfessionSkillUndefined,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSkills(makeDefRaws(tt.mutate))
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

	sword, ok := defs.Def(gc.SkillSword)
	require.True(t, ok)
	assert.Equal(t, gc.SkillDef{
		Name:     "Swordsmanship",
		Summary:  "Technique for handling swords",
		GainedBy: "Raised by attacking with swords",
		Effect:   "Increases sword damage and accuracy",
		Ability:  gc.AblSTR,
	}, sword)

	assert.Equal(t, []gc.SkillCategory{
		{Name: "Melee", IDs: []gc.SkillID{gc.SkillSword, gc.SkillSpear, gc.SkillFist}},
		{Name: "Ranged", IDs: []gc.SkillID{gc.SkillBow, gc.SkillHandgun, gc.SkillRifle, gc.SkillCannon}},
		{Name: "Craft", IDs: []gc.SkillID{gc.SkillCrafting, gc.SkillSmithing, gc.SkillNegotiation, gc.SkillMechanic}},
		{Name: "Mobility", IDs: []gc.SkillID{gc.SkillSprinting, gc.SkillStealth, gc.SkillNightVision, gc.SkillWeightBearing}},
		{Name: "Survival", IDs: []gc.SkillID{gc.SkillColdResist, gc.SkillHungerResist, gc.SkillHealing, gc.SkillExploration}},
		{Name: "Defense", IDs: []gc.SkillID{gc.SkillHeavyArmor, gc.SkillFireResist, gc.SkillThunderResist, gc.SkillChillResist, gc.SkillPhotonResist}},
	}, defs.Categories())

	skills := gc.NewSkills()
	for _, id := range gc.AllSkillIDs {
		skills.Get(id).Value = 10
	}
	got := map[gc.ProficiencyKey]int{}
	for _, key := range gc.ProficiencyKeys() {
		got[key] = int(defs.ProficiencyValue(skills, nil, gc.HealthyBodyFuncs(), key))
	}
	assert.Equal(t, map[gc.ProficiencyKey]int{
		gc.ProfFireResist: 70, gc.ProfThunderResist: 70, gc.ProfChillResist: 70, gc.ProfPhotonResist: 70,
		gc.ProfColdProgress: 70, gc.ProfHungerProgress: 80, gc.ProfHealingEffect: 150, gc.ProfMaxWeight: 140,
		gc.ProfExploration: 140, gc.ProfEnemyVision: 70, gc.ProfNightVision: 150, gc.ProfMoveCost: 80,
		gc.ProfCraftCost: 70, gc.ProfSmithQuality: 130, gc.ProfBuyPrice: 80, gc.ProfSellPrice: 120, gc.ProfHeavyArmor: 50,
		gc.ProfSwordDamage: 150, gc.ProfSpearDamage: 150, gc.ProfFistDamage: 150, gc.ProfBowDamage: 150,
		gc.ProfHandgunDamage: 150, gc.ProfRifleDamage: 150, gc.ProfCannonDamage: 150,
		gc.ProfSwordAccuracy: 130, gc.ProfSpearAccuracy: 130, gc.ProfFistAccuracy: 130, gc.ProfBowAccuracy: 130,
		gc.ProfHandgunAccuracy: 130, gc.ProfRifleAccuracy: 130, gc.ProfCannonAccuracy: 130,
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
			PainPerSeverity: 6, BodyFuncDropPerSeverity: 10, BloodDropPerSeverity: 22, Cause: gc.CauseFrozen, Onset: gc.OnsetCold,
		}},
		{gc.ConditionType("fracture"), gc.ConditionDef{
			Name: "Fracture", Description: "A broken bone. It will not heal until treated.",
			PainPerSeverity: 18, BodyFuncDropPerSeverity: 20, Recovery: gc.RecoverAfterTend, RecoverPer: 1,
			Parts: []gc.BodyPart{gc.BodyPartArms, gc.BodyPartHands, gc.BodyPartLegs, gc.BodyPartFeet},
		}},
		{gc.ConditionType("laceration"), gc.ConditionDef{
			Name: "Laceration", Description: "An open wound. It will not heal until treated.",
			PainPerSeverity: 8, BodyFuncDropPerSeverity: 8, Recovery: gc.RecoverAfterTend, RecoverPer: 2,
			BloodDropPerSeverity: 25, Cause: gc.CauseBloodLoss,
		}},
		{gc.ConditionType("liver_illness"), gc.ConditionDef{
			Name: "Liver illness", Description: "It worsens while untreated and drains HP when severe.",
			PainPerSeverity: 4, BodyFuncDropPerSeverity: 10, Recovery: gc.ProgressUntilTend, WorsenPer: 2, RecoverPer: 1,
			BloodDropPerSeverity: 25, Cause: gc.CauseIllness,
		}},
		{gc.ConditionType("food_poisoning"), gc.ConditionDef{
			Name: "Food poisoning", Description: "Nausea from bad food. It clears on its own over time.",
			PainPerSeverity: 5, BodyFuncDropPerSeverity: 12, Recovery: gc.RecoverOverTime, RecoverPer: 2, Onset: gc.OnsetRottenFood,
		}},
		{gc.ConditionType("exhaustion"), gc.ConditionDef{
			Name: "Exhaustion", Description: "Worn out from lack of rest. Sleep to recover.",
			BodyFuncDropPerSeverity: 10, Onset: gc.OnsetFatigue,
		}},
		{gc.ConditionType("malnutrition"), gc.ConditionDef{
			Name: "Malnutrition", Description: "Weakened by hunger. Eat to recover.",
			BodyFuncDropPerSeverity: 10, Onset: gc.OnsetHunger,
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
		names[bp] = defs.PartName(bp)
	}
	assert.Equal(t, [gc.BodyPartCount]float64{2, 4, 3, 1, 3, 1, 0}, weights)
	assert.Equal(t, [gc.BodyPartCount]string{"Head", "Torso", "Arm", "Hand", "Leg", "Foot", "Whole body"}, names)
}
