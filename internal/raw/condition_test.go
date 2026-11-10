package raw

import (
	"fmt"
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaEnum は OpenAPI スキーマの enum の値を返す
func schemaEnum(name string) []string {
	spec, err := loadSpec()
	if err != nil {
		panic(err)
	}
	enum := spec.Components.Schemas[name].Value.Enum
	values := make([]string, 0, len(enum))
	for _, v := range enum {
		s, ok := v.(string)
		if !ok {
			panic(fmt.Sprintf("%s enum value %v is not string", name, v))
		}
		values = append(values, s)
	}
	return values
}

// makeDefRaws は validateCodeRows を通る raws を作り、mutate で崩す
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
	attackIDs := schemaEnum("AttackCategory")
	attackTypes := make([]oapi.AttackType, 0, len(attackIDs))
	for _, id := range attackIDs {
		attackTypes = append(attackTypes, oapi.AttackType{
			Id:       oapi.AttackCategory(id),
			Name:     id,
			Range:    oapi.AttackRangeMelee,
			Injury:   string(gc.ConditionType("fracture")),
			Skill:    oapi.SkillId(id),
			Damage:   oapi.ProficiencyId(id + "_damage"),
			Accuracy: oapi.ProficiencyId(id + "_accuracy"),
		})
	}
	skillIDs := schemaEnum("SkillId")
	skills := make([]oapi.Skill, 0, len(skillIDs))
	for _, id := range skillIDs {
		skills = append(skills, oapi.Skill{Id: oapi.SkillId(id), Name: id, Ability: oapi.Str, Category: "general", Proficiencies: &[]oapi.SkillProficiency{}})
	}
	resists := map[string]oapi.Element{
		"fire_resist": oapi.ElementFIRE, "thunder_resist": oapi.ElementTHUNDER,
		"chill_resist": oapi.ElementCHILL, "photon_resist": oapi.ElementPHOTON,
	}
	for _, key := range schemaEnum("ProficiencyId") {
		owner := 0
		for i := range attackTypes {
			if oapi.ProficiencyId(key) == attackTypes[i].Damage || oapi.ProficiencyId(key) == attackTypes[i].Accuracy {
				owner = slices.Index(skillIDs, string(attackTypes[i].Skill))
			}
		}
		proficiency := oapi.SkillProficiency{Id: oapi.ProficiencyId(key), PerLevel: 1}
		if e, ok := resists[key]; ok {
			proficiency.Resists = &e
		}
		*skills[owner].Proficiencies = append(*skills[owner].Proficiencies, proficiency)
	}
	raws := oapi.Raws{
		BodyParts:       &bodyParts,
		Conditions:      &conditions,
		AttackTypes:     &attackTypes,
		Skills:          &skills,
		SkillCategories: &[]oapi.SkillCategory{{Id: "general", Name: "General"}},
		Items: &[]oapi.Item{
			{Id: "splint", Remedy: &oapi.Remedy{Treats: []oapi.ConditionTypeName{string(gc.ConditionType("fracture"))}}},
			{Id: "manual", Book: &oapi.Book{Skill: &oapi.SkillBook{TargetSkill: oapi.SkillIdSword}}},
		},
		Professions:     &[]oapi.Profession{{Id: "soldier", Skills: &[]oapi.ProfessionSkill{{Id: oapi.SkillIdRifle}}}},
		InteriorProfile: &oapi.InteriorProfile{},
		Weathers:        &[]oapi.Weather{{Id: oapi.Clear}},
		Members:         &[]oapi.Member{{Id: "ash", Player: new(true)}},
		ShelterEffects:  &[]oapi.ShelterEffect{{Id: oapi.Outdoor}, {Id: oapi.Partial}, {Id: oapi.Indoor}},
		WaterEffects:    &[]oapi.WaterEffect{{Id: oapi.Nearby}, {Id: oapi.Submerged}},
		FoliageEffects:  &[]oapi.FoliageEffect{{Id: oapi.Grass}, {Id: oapi.Forest}},
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
			name: "プレイヤーの member が無い",
			mutate: func(r *oapi.Raws) {
				(*r.Members)[0].Player = nil
			},
			wantErr: errPlayerMemberMissing,
		},
		{
			name: "プレイヤーの member が2つある",
			mutate: func(r *oapi.Raws) {
				*r.Members = append(*r.Members, oapi.Member{Id: "ash2", Player: new(true)})
			},
			wantErr: errPlayerMemberDuplicate,
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
			name: "近接の攻撃種別が射程を持つ",
			mutate: func(r *oapi.Raws) {
				(*r.AttackTypes)[0].MaxRange = new(5)
			},
			wantErr: errAttackRangeParams,
		},
		{
			name: "遠距離の攻撃種別が射程を持たない",
			mutate: func(r *oapi.Raws) {
				(*r.AttackTypes)[0].Range = oapi.AttackRangeRanged
			},
			wantErr: errAttackRangeParams,
		},
		{
			name: "攻撃種別の行が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.AttackTypes = (*r.AttackTypes)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "スキルの行が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.Skills = (*r.Skills)[:len(*r.Skills)-1]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "熟練の効果倍率の行が欠ける",
			mutate: func(r *oapi.Raws) {
				*(*r.Skills)[0].Proficiencies = (*(*r.Skills)[0].Proficiencies)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "攻撃属性を軽減する倍率が欠ける",
			mutate: func(r *oapi.Raws) {
				for i := range *r.Skills {
					for j := range *(*r.Skills)[i].Proficiencies {
						p := &(*(*r.Skills)[i].Proficiencies)[j]
						if p.Id == oapi.ProficiencyIdChillResist {
							p.Resists = nil
						}
					}
				}
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "無属性を軽減する倍率",
			mutate: func(r *oapi.Raws) {
				none := oapi.ElementNONE
				for i := range *r.Skills {
					for j := range *(*r.Skills)[i].Proficiencies {
						p := &(*(*r.Skills)[i].Proficiencies)[j]
						if p.Id == oapi.ProficiencyIdChillResist {
							p.Resists = &none
						}
					}
				}
			},
			wantErr: errResistNone,
		},
		{
			name: "武器の効果倍率を別のスキルが持つ",
			mutate: func(r *oapi.Raws) {
				sword := slices.Index(schemaEnum("SkillId"), string(oapi.SkillIdSword))
				spear := slices.Index(schemaEnum("SkillId"), string(oapi.SkillIdSpear))
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

	sword, ok := defs.Skills[oapi.SkillIdSword]
	require.True(t, ok)
	assert.Equal(t, gc.SkillDef{
		Name:     "Swordsmanship",
		Summary:  "Technique for handling swords",
		GainedBy: "Raised by attacking with swords",
		Effect:   "Increases sword damage and accuracy",
		Ability:  oapi.Str,
	}, sword)

	assert.Equal(t, []gc.SkillCategory{
		{Name: "Melee", IDs: []oapi.SkillId{oapi.SkillIdSword, oapi.SkillIdSpear, oapi.SkillIdFist}},
		{Name: "Ranged", IDs: []oapi.SkillId{oapi.SkillIdBow, oapi.SkillIdHandgun, oapi.SkillIdRifle, oapi.SkillIdCannon}},
		{Name: "Craft", IDs: []oapi.SkillId{oapi.SkillIdCrafting, oapi.SkillIdSmithing, oapi.SkillIdNegotiation, oapi.SkillIdMechanic}},
		{Name: "Mobility", IDs: []oapi.SkillId{oapi.SkillIdSprinting, oapi.SkillIdStealth, oapi.SkillIdNightVision, oapi.SkillIdWeightBearing}},
		{Name: "Survival", IDs: []oapi.SkillId{oapi.SkillIdColdResist, oapi.SkillIdHungerResist, oapi.SkillIdHealing, oapi.SkillIdExploration}},
		{Name: "Defense", IDs: []oapi.SkillId{oapi.SkillIdHeavyArmor, oapi.SkillIdFireResist, oapi.SkillIdThunderResist, oapi.SkillIdChillResist, oapi.SkillIdPhotonResist}},
	}, defs.Categories)

	weapons := map[string][3]string{}
	accuracyLimbs := map[string]gc.BodyFuncKind{}
	for _, w := range defs.Weapons {
		weapons[string(w.AttackType)] = [3]string{string(w.Skill), string(w.Damage), string(w.Accuracy)}
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
	}, weapons)
	assert.Equal(t, map[string]gc.BodyFuncKind{
		"sword": gc.BodyFuncManipulation, "spear": gc.BodyFuncManipulation, "fist": gc.BodyFuncManipulation,
		"bow": gc.BodyFuncSight, "handgun": gc.BodyFuncSight, "rifle": gc.BodyFuncSight, "cannon": gc.BodyFuncSight,
	}, accuracyLimbs, "命中は近接なら操作、遠隔なら視覚で目減りする")
	assert.Equal(t, gc.BodyFuncKind(""), defs.Proficiencies[oapi.ProficiencyIdSwordDamage].BodyFunc, "ダメージは身体機能で目減りしない")
	assert.Equal(t, []gc.ResistDef{
		{Element: oapi.ElementFIRE, Proficiency: oapi.ProficiencyIdFireResist},
		{Element: oapi.ElementTHUNDER, Proficiency: oapi.ProficiencyIdThunderResist},
		{Element: oapi.ElementCHILL, Proficiency: oapi.ProficiencyIdChillResist},
		{Element: oapi.ElementPHOTON, Proficiency: oapi.ProficiencyIdPhotonResist},
	}, defs.Resists)

	skills := gc.NewSkills()
	for _, id := range schemaEnum("SkillId") {
		skills.Get(oapi.SkillId(id)).Value = 10
	}
	got := map[oapi.ProficiencyId]int{}
	for _, key := range schemaEnum("ProficiencyId") {
		got[oapi.ProficiencyId(key)] = int(defs.ProficiencyValue(skills, nil, gc.HealthyBodyFuncs(), oapi.ProficiencyId(key)))
	}
	assert.Equal(t, map[oapi.ProficiencyId]int{
		oapi.ProficiencyIdFireResist: 70, oapi.ProficiencyIdThunderResist: 70, oapi.ProficiencyIdChillResist: 70, oapi.ProficiencyIdPhotonResist: 70,
		oapi.ProficiencyIdColdProgress: 70, oapi.ProficiencyIdHungerProgress: 80, oapi.ProficiencyIdHealingEffect: 150, oapi.ProficiencyIdMaxWeight: 140,
		oapi.ProficiencyIdExploration: 140, oapi.ProficiencyIdEnemyVision: 70, oapi.ProficiencyIdNightVision: 150, oapi.ProficiencyIdMoveCost: 80,
		oapi.ProficiencyIdCraftCost: 70, oapi.ProficiencyIdSmithQuality: 130, oapi.ProficiencyIdBuyPrice: 80, oapi.ProficiencyIdSellPrice: 120, oapi.ProficiencyIdHeavyArmor: 50,
		oapi.ProficiencyIdSwordDamage: 150, oapi.ProficiencyIdSpearDamage: 150, oapi.ProficiencyIdFistDamage: 150, oapi.ProficiencyIdBowDamage: 150,
		oapi.ProficiencyIdHandgunDamage: 150, oapi.ProficiencyIdRifleDamage: 150, oapi.ProficiencyIdCannonDamage: 150,
		oapi.ProficiencyIdSwordAccuracy: 130, oapi.ProficiencyIdSpearAccuracy: 130, oapi.ProficiencyIdFistAccuracy: 130, oapi.ProficiencyIdBowAccuracy: 130,
		oapi.ProficiencyIdHandgunAccuracy: 130, oapi.ProficiencyIdRifleAccuracy: 130, oapi.ProficiencyIdCannonAccuracy: 130,
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
			PainPerSeverity: 6, BodyFuncDropPerSeverity: 10, BloodDropPerSeverity: 22, Cause: gc.CauseFrozen, Onset: oapi.Cold,
		}},
		{gc.ConditionType("fracture"), gc.ConditionDef{
			Name: "Fracture", Description: "A broken bone. It will not heal until treated.",
			PainPerSeverity: 18, BodyFuncDropPerSeverity: 20, Recovery: oapi.RecoverAfterTend, RecoverPer: 1,
			Parts: []gc.BodyPart{gc.BodyPartArms, gc.BodyPartHands, gc.BodyPartLegs, gc.BodyPartFeet},
		}},
		{gc.ConditionType("laceration"), gc.ConditionDef{
			Name: "Laceration", Description: "An open wound. It will not heal until treated.",
			PainPerSeverity: 8, BodyFuncDropPerSeverity: 8, Recovery: oapi.RecoverAfterTend, RecoverPer: 2,
			BloodDropPerSeverity: 25, Cause: gc.CauseBloodLoss,
		}},
		{gc.ConditionType("liver_illness"), gc.ConditionDef{
			Name: "Liver illness", Description: "It worsens while untreated and drains HP when severe.",
			PainPerSeverity: 4, BodyFuncDropPerSeverity: 10, Recovery: oapi.ProgressUntilTend, WorsenPer: 2, RecoverPer: 1,
			BloodDropPerSeverity: 25, Cause: gc.CauseIllness,
		}},
		{gc.ConditionType("food_poisoning"), gc.ConditionDef{
			Name: "Food poisoning", Description: "Nausea from bad food. It clears on its own over time.",
			PainPerSeverity: 5, BodyFuncDropPerSeverity: 12, Recovery: oapi.RecoverOverTime, RecoverPer: 2, Onset: oapi.RottenFood,
		}},
		{gc.ConditionType("exhaustion"), gc.ConditionDef{
			Name: "Exhaustion", Description: "Worn out from lack of rest. Sleep to recover.",
			BodyFuncDropPerSeverity: 10, Onset: oapi.Fatigue,
		}},
		{gc.ConditionType("malnutrition"), gc.ConditionDef{
			Name: "Malnutrition", Description: "Weakened by hunger. Eat to recover.",
			BodyFuncDropPerSeverity: 10, Onset: oapi.Hunger,
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
