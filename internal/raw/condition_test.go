package raw

import (
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeDefRaws は validateCodeRows を通る raws を作り、mutate で崩す
// enumRowsOf は enum の値 ids それぞれに1行ずつ作る
func enumRowsOf[R any](ids []string, row func(string) R) []R {
	out := make([]R, 0, len(ids))
	for _, id := range ids {
		out = append(out, row(id))
	}
	return out
}

func makeDefRaws(mutate func(*rawschema.Raws)) rawschema.Raws {
	bodyParts := make([]rawschema.BodyPart, 0, gc.BodyPartCount)
	for bp := range gc.BodyPartCount {
		bodyParts = append(bodyParts, rawschema.BodyPart{Id: bp.ID(), Name: bp.ID(), HitWeight: 1})
	}
	onset := func(o rawschema.ConditionOnset) *rawschema.ConditionOnset { return &o }
	conditions := []rawschema.Condition{
		{Id: "hypothermia", Name: "Hypothermia", Description: "Hypothermia", Onset: onset(rawschema.Cold)},
		{Id: "fracture", Name: "Fracture", Description: "Fracture"},
		{Id: "food_poisoning", Name: "Food poisoning", Description: "Food poisoning", Onset: onset(rawschema.RottenFood)},
		{Id: "exhaustion", Name: "Exhaustion", Description: "Exhaustion", Onset: onset(rawschema.Fatigue)},
		{Id: "malnutrition", Name: "Malnutrition", Description: "Malnutrition", Onset: onset(rawschema.Hunger)},
	}
	attackIDs := enumStrings(rawschema.AttackCategoryValues())
	attackTypes := make([]rawschema.AttackType, 0, len(attackIDs))
	for _, id := range attackIDs {
		attackTypes = append(attackTypes, rawschema.AttackType{
			Id:       rawschema.AttackCategory(id),
			Name:     id,
			Range:    rawschema.AttackRangeMelee,
			Injury:   string(gc.ConditionType("fracture")),
			Skill:    rawschema.SkillId(id),
			Damage:   rawschema.ProficiencyId(id + "_damage"),
			Accuracy: rawschema.ProficiencyId(id + "_accuracy"),
		})
	}
	skillIDs := enumStrings(rawschema.SkillIdValues())
	skills := make([]rawschema.Skill, 0, len(skillIDs))
	for _, id := range skillIDs {
		skills = append(skills, rawschema.Skill{Id: rawschema.SkillId(id), Name: id, Ability: rawschema.Str, Category: "general", Proficiencies: &[]rawschema.SkillProficiency{}})
	}
	resists := map[string]rawschema.Element{
		"fire_resist": rawschema.ElementFIRE, "thunder_resist": rawschema.ElementTHUNDER,
		"chill_resist": rawschema.ElementCHILL, "photon_resist": rawschema.ElementPHOTON,
	}
	for _, key := range enumStrings(rawschema.ProficiencyIdValues()) {
		owner := 0
		for i := range attackTypes {
			if rawschema.ProficiencyId(key) == attackTypes[i].Damage || rawschema.ProficiencyId(key) == attackTypes[i].Accuracy {
				owner = slices.Index(skillIDs, string(attackTypes[i].Skill))
			}
		}
		proficiency := rawschema.SkillProficiency{Id: rawschema.ProficiencyId(key), PerLevel: 1}
		if e, ok := resists[key]; ok {
			proficiency.Resists = &e
		}
		*skills[owner].Proficiencies = append(*skills[owner].Proficiencies, proficiency)
	}
	freshness := enumRowsOf(enumStrings(rawschema.FreshnessStageIdValues()), func(id string) rawschema.FreshnessStage {
		return rawschema.FreshnessStage{Id: rawschema.FreshnessStageId(id)}
	})
	materials := enumRowsOf(enumStrings(rawschema.MaterialValues()), func(id string) rawschema.MaterialDef { return rawschema.MaterialDef{Id: rawschema.Material(id)} })
	tools := enumRowsOf(enumStrings(rawschema.ToolCategoryValues()), func(id string) rawschema.ToolCategoryDef {
		return rawschema.ToolCategoryDef{Id: rawschema.ToolCategory(id)}
	})
	timesOfDay := enumRowsOf(enumStrings(rawschema.TimeOfDayIdValues()), func(id string) rawschema.TimeOfDay { return rawschema.TimeOfDay{Id: rawschema.TimeOfDayId(id)} })
	seasons := enumRowsOf(enumStrings(rawschema.SeasonIdValues()), func(id string) rawschema.Season { return rawschema.Season{Id: rawschema.SeasonId(id)} })
	raws := rawschema.Raws{
		Dungeons:        &[]rawschema.Dungeon{{Id: "dead_forest"}},
		Settlements:     &[]rawschema.Settlement{{Id: "village", Weight: 1}},
		TimesOfDay:      &timesOfDay,
		Seasons:         &seasons,
		FreshnessStages: &freshness,
		Materials:       &materials,
		ToolCategories:  &tools,
		BodyParts:       &bodyParts,
		Conditions:      &conditions,
		AttackTypes:     &attackTypes,
		Skills:          &skills,
		SkillCategories: &[]rawschema.SkillCategory{{Id: "general", Name: "General"}},
		Items: &[]rawschema.Item{
			{Id: "splint", Remedy: &rawschema.Remedy{Treats: []rawschema.ConditionTypeName{string(gc.ConditionType("fracture"))}}},
			{Id: "manual", Book: &rawschema.Book{Skill: &rawschema.SkillBook{TargetSkill: rawschema.SkillIdSword}}},
		},
		Professions:     &[]rawschema.Profession{{Id: "soldier", Skills: &[]rawschema.ProfessionSkill{{Id: rawschema.SkillIdRifle}}}},
		InteriorProfile: &rawschema.InteriorProfile{},
		Weathers:        &[]rawschema.Weather{{Id: rawschema.Clear}},
		Members:         &[]rawschema.Member{{Id: "ash", Player: new(true)}},
		ShelterEffects:  &[]rawschema.ShelterEffect{{Id: rawschema.Outdoor}, {Id: rawschema.Partial}, {Id: rawschema.Indoor}},
		WaterEffects:    &[]rawschema.WaterEffect{{Id: rawschema.Nearby}, {Id: rawschema.Submerged}},
		FoliageEffects:  &[]rawschema.FoliageEffect{{Id: rawschema.FoliageGrass}, {Id: rawschema.FoliageForest}},
		HungerStages:    &[]rawschema.HungerStage{{Id: rawschema.HungerStageIdStarving}},
		FatigueStages:   &[]rawschema.FatigueStage{{Id: rawschema.FatigueStageIdRested}},
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
			name:   "全行がそろい参照が実在すれば通る",
			mutate: func(*rawschema.Raws) {},
		},
		{
			name: "部位を限った不調は命中しうる部位があれば通る",
			mutate: func(r *rawschema.Raws) {
				(*r.Conditions)[0].Parts = &[]rawschema.BodyPartId{"arm", "leg"}
			},
		},
		{
			name: "部位の id 重複",
			mutate: func(r *rawschema.Raws) {
				*r.BodyParts = append(*r.BodyParts, rawschema.BodyPart{Id: "arm", Name: "Arm"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "不調の id 重複",
			mutate: func(r *rawschema.Raws) {
				*r.Conditions = append(*r.Conditions, rawschema.Condition{Id: "fracture"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "Go に無い部位の行",
			mutate: func(r *rawschema.Raws) {
				*r.BodyParts = append(*r.BodyParts, rawschema.BodyPart{Id: "tail", Name: "Tail"})
			},
			wantErr: errBodyPartUnknown,
		},
		{
			name: "不調の部位が実在しない",
			mutate: func(r *rawschema.Raws) {
				(*r.Conditions)[0].Parts = &[]rawschema.BodyPartId{"tail"}
			},
			wantErr: errConditionPartUndefined,
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
			name: "全部位の不調でも命中する部位が無い",
			mutate: func(r *rawschema.Raws) {
				for i := range *r.BodyParts {
					(*r.BodyParts)[i].HitWeight = 0
				}
			},
			wantErr: errConditionNoHitPart,
		},
		{
			name: "攻撃種別の怪我が実在しない",
			mutate: func(r *rawschema.Raws) {
				(*r.AttackTypes)[0].Injury = "bruise"
			},
			wantErr: errAttackTypeInjuryUndefined,
		},
		{
			name: "攻撃種別の id 重複",
			mutate: func(r *rawschema.Raws) {
				*r.AttackTypes = append(*r.AttackTypes, (*r.AttackTypes)[0])
			},
			wantErr: errDuplicateID,
		},
		{
			name: "治療の対象が実在しない",
			mutate: func(r *rawschema.Raws) {
				(*r.Items)[0].Remedy.Treats = []rawschema.ConditionTypeName{"Fracture"}
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
			name: "仕組みを持つ不調が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.Conditions = (*r.Conditions)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "同じ仕組みを2つの不調が持つ",
			mutate: func(r *rawschema.Raws) {
				(*r.Conditions)[1].Onset = (*r.Conditions)[0].Onset
			},
			wantErr: errDuplicateID,
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
			name: "攻撃種別の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.AttackTypes = (*r.AttackTypes)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "スキルの行が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.Skills = (*r.Skills)[:len(*r.Skills)-1]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "熟練の効果倍率の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				*(*r.Skills)[0].Proficiencies = (*(*r.Skills)[0].Proficiencies)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "攻撃属性を軽減する倍率が欠ける",
			mutate: func(r *rawschema.Raws) {
				for i := range *r.Skills {
					for j := range *(*r.Skills)[i].Proficiencies {
						p := &(*(*r.Skills)[i].Proficiencies)[j]
						if p.Id == rawschema.ProficiencyIdChillResist {
							p.Resists = nil
						}
					}
				}
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "無属性を軽減する倍率",
			mutate: func(r *rawschema.Raws) {
				none := rawschema.ElementNONE
				for i := range *r.Skills {
					for j := range *(*r.Skills)[i].Proficiencies {
						p := &(*(*r.Skills)[i].Proficiencies)[j]
						if p.Id == rawschema.ProficiencyIdChillResist {
							p.Resists = &none
						}
					}
				}
			},
			wantErr: errResistNone,
		},
		{
			name: "武器の効果倍率を別のスキルが持つ",
			mutate: func(r *rawschema.Raws) {
				sword := slices.Index(enumStrings(rawschema.SkillIdValues()), string(rawschema.SkillIdSword))
				spear := slices.Index(enumStrings(rawschema.SkillIdValues()), string(rawschema.SkillIdSpear))
				(*r.Skills)[sword].Proficiencies, (*r.Skills)[spear].Proficiencies = (*r.Skills)[spear].Proficiencies, (*r.Skills)[sword].Proficiencies
			},
			wantErr: errWeaponProficiencyOwner,
		},
		{
			name: "鮮度の段階の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.FreshnessStages = (*r.FreshnessStages)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "材質の行が重複する",
			mutate: func(r *rawschema.Raws) {
				*r.Materials = append(*r.Materials, rawschema.MaterialDef{Id: rawschema.WOOD})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "道具の分類の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				r.ToolCategories = nil
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "時間帯の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.TimesOfDay = (*r.TimesOfDay)[:len(*r.TimesOfDay)-1]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "季節の行が重複する",
			mutate: func(r *rawschema.Raws) {
				*r.Seasons = append(*r.Seasons, rawschema.Season{Id: rawschema.Winter})
			},
			wantErr: errDuplicateID,
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
			name: "生活感の表が無い",
			mutate: func(r *rawschema.Raws) {
				r.InteriorProfile = nil
			},
			wantErr: errTableMissing,
		},
		{
			name: "天候の表が空",
			mutate: func(r *rawschema.Raws) {
				r.Weathers = &[]rawschema.Weather{}
			},
			wantErr: errTableMissing,
		},
		{
			name: "疲労の段階が無い",
			mutate: func(r *rawschema.Raws) {
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
		mutate  func(*rawschema.Raws)
		wantErr error
	}{
		{
			name:   "スキルと参照が実在すれば通る",
			mutate: func(*rawschema.Raws) {},
		},
		{
			name: "スキルの id 重複",
			mutate: func(r *rawschema.Raws) {
				*r.Skills = append(*r.Skills, rawschema.Skill{Id: "sword", Category: "general"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "カテゴリの id 重複",
			mutate: func(r *rawschema.Raws) {
				*r.SkillCategories = append(*r.SkillCategories, rawschema.SkillCategory{Id: "general"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "熟練の効果倍率の id がスキルをまたいで重複",
			mutate: func(r *rawschema.Raws) {
				*(*r.Skills)[1].Proficiencies = append(*(*r.Skills)[1].Proficiencies, (*(*r.Skills)[0].Proficiencies)[0])
			},
			wantErr: errDuplicateID,
		},
		{
			name: "スキルのカテゴリが実在しない",
			mutate: func(r *rawschema.Raws) {
				(*r.Skills)[0].Category = "magic"
			},
			wantErr: errSkillCategoryUndefined,
		},
		{
			name: "本のスキルが実在しない",
			mutate: func(r *rawschema.Raws) {
				(*r.Items)[1].Book.Skill.TargetSkill = "juggling"
			},
			wantErr: errBookSkillUndefined,
		},
		{
			name: "職業のスキルが実在しない",
			mutate: func(r *rawschema.Raws) {
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

	sword, ok := defs.Skills[rawschema.SkillIdSword]
	require.True(t, ok)
	assert.Equal(t, gc.SkillDef{
		Name:     "Swordsmanship",
		Summary:  "Technique for handling swords",
		GainedBy: "Raised by attacking with swords",
		Effect:   "Increases sword damage and accuracy",
		Ability:  rawschema.Str,
	}, sword)

	assert.Equal(t, []gc.SkillCategory{
		{Name: "Melee", IDs: []rawschema.SkillId{rawschema.SkillIdSword, rawschema.SkillIdSpear, rawschema.SkillIdFist}},
		{Name: "Ranged", IDs: []rawschema.SkillId{rawschema.SkillIdBow, rawschema.SkillIdHandgun, rawschema.SkillIdRifle, rawschema.SkillIdCannon}},
		{Name: "Craft", IDs: []rawschema.SkillId{rawschema.SkillIdCrafting, rawschema.SkillIdSmithing, rawschema.SkillIdNegotiation, rawschema.SkillIdMechanic}},
		{Name: "Mobility", IDs: []rawschema.SkillId{rawschema.SkillIdSprinting, rawschema.SkillIdStealth, rawschema.SkillIdNightVision, rawschema.SkillIdWeightBearing}},
		{Name: "Survival", IDs: []rawschema.SkillId{rawschema.SkillIdColdResist, rawschema.SkillIdHungerResist, rawschema.SkillIdHealing, rawschema.SkillIdExploration}},
		{Name: "Defense", IDs: []rawschema.SkillId{rawschema.SkillIdHeavyArmor, rawschema.SkillIdFireResist, rawschema.SkillIdThunderResist, rawschema.SkillIdChillResist, rawschema.SkillIdPhotonResist}},
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
	assert.Equal(t, gc.BodyFuncKind(""), defs.Proficiencies[rawschema.ProficiencyIdSwordDamage].BodyFunc, "ダメージは身体機能で目減りしない")
	assert.Equal(t, []gc.ResistDef{
		{Element: rawschema.ElementFIRE, Proficiency: rawschema.ProficiencyIdFireResist},
		{Element: rawschema.ElementTHUNDER, Proficiency: rawschema.ProficiencyIdThunderResist},
		{Element: rawschema.ElementCHILL, Proficiency: rawschema.ProficiencyIdChillResist},
		{Element: rawschema.ElementPHOTON, Proficiency: rawschema.ProficiencyIdPhotonResist},
	}, defs.Resists)

	skills := gc.NewSkills()
	for _, id := range enumStrings(rawschema.SkillIdValues()) {
		skills.Get(rawschema.SkillId(id)).Value = 10
	}
	got := map[rawschema.ProficiencyId]int{}
	for _, key := range enumStrings(rawschema.ProficiencyIdValues()) {
		got[rawschema.ProficiencyId(key)] = int(defs.ProficiencyValue(skills, nil, gc.HealthyBodyFuncs(), rawschema.ProficiencyId(key)))
	}
	assert.Equal(t, map[rawschema.ProficiencyId]int{
		rawschema.ProficiencyIdFireResist: 70, rawschema.ProficiencyIdThunderResist: 70, rawschema.ProficiencyIdChillResist: 70, rawschema.ProficiencyIdPhotonResist: 70,
		rawschema.ProficiencyIdColdProgress: 70, rawschema.ProficiencyIdHungerProgress: 80, rawschema.ProficiencyIdHealingEffect: 150, rawschema.ProficiencyIdMaxWeight: 140,
		rawschema.ProficiencyIdExploration: 140, rawschema.ProficiencyIdEnemyVision: 70, rawschema.ProficiencyIdNightVision: 150, rawschema.ProficiencyIdMoveCost: 80,
		rawschema.ProficiencyIdCraftCost: 70, rawschema.ProficiencyIdSmithQuality: 130, rawschema.ProficiencyIdBuyPrice: 80, rawschema.ProficiencyIdSellPrice: 120, rawschema.ProficiencyIdHeavyArmor: 50,
		rawschema.ProficiencyIdSwordDamage: 150, rawschema.ProficiencyIdSpearDamage: 150, rawschema.ProficiencyIdFistDamage: 150, rawschema.ProficiencyIdBowDamage: 150,
		rawschema.ProficiencyIdHandgunDamage: 150, rawschema.ProficiencyIdRifleDamage: 150, rawschema.ProficiencyIdCannonDamage: 150,
		rawschema.ProficiencyIdSwordAccuracy: 130, rawschema.ProficiencyIdSpearAccuracy: 130, rawschema.ProficiencyIdFistAccuracy: 130, rawschema.ProficiencyIdBowAccuracy: 130,
		rawschema.ProficiencyIdHandgunAccuracy: 130, rawschema.ProficiencyIdRifleAccuracy: 130, rawschema.ProficiencyIdCannonAccuracy: 130,
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
			PainPerSeverity: 6, BodyFuncDropPerSeverity: 10, BloodDropPerSeverity: 22, Cause: gc.CauseFrozen, Onset: rawschema.Cold,
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
func mapKeys[K ~string, V any](m map[K]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, string(k))
	}
	return out
}

func TestOrdinalEnumMaps_スキーマのenumの全値を持つ(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, enumStrings(rawschema.WeatherIdValues()), mapKeys(weatherKinds))
	assert.ElementsMatch(t, enumStrings(rawschema.SeasonIdValues()), mapKeys(seasons))
	assert.ElementsMatch(t, enumStrings(rawschema.TimeOfDayIdValues()), mapKeys(timesOfDay))
	assert.ElementsMatch(t, enumStrings(rawschema.ConditionSeverityValues()), mapKeys(severities))
}
