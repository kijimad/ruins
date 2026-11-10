package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

// testSkillDefs は components のテストが使うスキルの定義の表。値は raw の行と同じにする
var testSkillDefs = func() SkillDefs {
	abilities := map[AbilityID][]oapi.SkillId{
		AblSTR: {oapi.SkillIdSword, oapi.SkillIdSpear, oapi.SkillIdFist, oapi.SkillIdWeightBearing},
		AblSEN: {oapi.SkillIdBow, oapi.SkillIdHandgun, oapi.SkillIdRifle, oapi.SkillIdCannon, oapi.SkillIdExploration},
		AblDEX: {oapi.SkillIdCrafting, oapi.SkillIdSmithing, oapi.SkillIdNegotiation, oapi.SkillIdMechanic},
		AblAGI: {oapi.SkillIdSprinting, oapi.SkillIdStealth, oapi.SkillIdNightVision},
		AblVIT: {oapi.SkillIdColdResist, oapi.SkillIdHungerResist, oapi.SkillIdHealing},
		AblDEF: {oapi.SkillIdHeavyArmor, oapi.SkillIdFireResist, oapi.SkillIdThunderResist, oapi.SkillIdChillResist, oapi.SkillIdPhotonResist},
	}
	skills := map[oapi.SkillId]SkillDef{}
	for abl, ids := range abilities {
		for _, id := range ids {
			skills[id] = SkillDef{Name: string(id), Ability: abl}
		}
	}
	proficiencies := map[oapi.ProficiencyId]ProficiencyDef{
		oapi.ProficiencyIdFireResist:     {Skill: oapi.SkillIdFireResist, PerLevel: -3},
		oapi.ProficiencyIdThunderResist:  {Skill: oapi.SkillIdThunderResist, PerLevel: -3},
		oapi.ProficiencyIdChillResist:    {Skill: oapi.SkillIdChillResist, PerLevel: -3},
		oapi.ProficiencyIdPhotonResist:   {Skill: oapi.SkillIdPhotonResist, PerLevel: -3},
		oapi.ProficiencyIdColdProgress:   {Skill: oapi.SkillIdColdResist, PerLevel: -3},
		oapi.ProficiencyIdHungerProgress: {Skill: oapi.SkillIdHungerResist, PerLevel: -2},
		oapi.ProficiencyIdHealingEffect:  {Skill: oapi.SkillIdHealing, PerLevel: 5},
		oapi.ProficiencyIdMaxWeight:      {Skill: oapi.SkillIdWeightBearing, PerLevel: 4},
		oapi.ProficiencyIdExploration:    {Skill: oapi.SkillIdExploration, PerLevel: 4},
		oapi.ProficiencyIdEnemyVision:    {Skill: oapi.SkillIdStealth, PerLevel: -3},
		oapi.ProficiencyIdNightVision:    {Skill: oapi.SkillIdNightVision, PerLevel: 5},
		oapi.ProficiencyIdMoveCost:       {Skill: oapi.SkillIdSprinting, PerLevel: -2},
		oapi.ProficiencyIdCraftCost:      {Skill: oapi.SkillIdCrafting, PerLevel: -3},
		oapi.ProficiencyIdSmithQuality:   {Skill: oapi.SkillIdSmithing, PerLevel: 3},
		oapi.ProficiencyIdBuyPrice:       {Skill: oapi.SkillIdNegotiation, PerLevel: -2},
		oapi.ProficiencyIdSellPrice:      {Skill: oapi.SkillIdNegotiation, PerLevel: 2},
		oapi.ProficiencyIdHeavyArmor:     {Skill: oapi.SkillIdHeavyArmor, PerLevel: -5},
	}
	attackTypes := []AttackType{AttackSword, AttackSpear, AttackFist, AttackBow, AttackHandgun, AttackRifle, AttackCannon}
	weapons := make([]WeaponDef, 0, len(attackTypes))
	for _, at := range attackTypes {
		w := WeaponDef{
			AttackType: at,
			Skill:      oapi.SkillId(at.Type),
			Damage:     oapi.ProficiencyId(at.Type + "_damage"),
			Accuracy:   oapi.ProficiencyId(at.Type + "_accuracy"),
		}
		weapons = append(weapons, w)
		limb := BodyFuncManipulation
		if at.Range == AttackRangeRanged {
			limb = BodyFuncSight
		}
		proficiencies[w.Damage] = ProficiencyDef{Skill: w.Skill, PerLevel: 5}
		proficiencies[w.Accuracy] = ProficiencyDef{Skill: w.Skill, PerLevel: 3, BodyFunc: limb}
	}
	categories := []SkillCategory{{Name: "Melee", IDs: []oapi.SkillId{oapi.SkillIdSword, oapi.SkillIdSpear, oapi.SkillIdFist}}}
	return SkillDefs{Skills: skills, Categories: categories, Proficiencies: proficiencies, Weapons: weapons}
}()

func TestSkillDefs_Ability(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AblSTR, testSkillDefs.Ability(oapi.SkillIdSword))
	assert.Equal(t, AblDEF, testSkillDefs.Ability(oapi.SkillIdFireResist))
	assert.Panics(t, func() { testSkillDefs.Ability("undefined_skill") }, "未登録は panic する")
}

func TestSkillDefs_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "sword", testSkillDefs.Name(oapi.SkillIdSword))
	assert.Equal(t, "undefined_skill", testSkillDefs.Name("undefined_skill"), "未登録は id をそのまま返す")
}

func TestSkillDefs_Weapon(t *testing.T) {
	t.Parallel()

	got, ok := testSkillDefs.Weapon(AttackBow)
	assert.True(t, ok)
	assert.Equal(t, WeaponDef{AttackType: AttackBow, Skill: oapi.SkillIdBow, Damage: oapi.ProficiencyIdBowDamage, Accuracy: oapi.ProficiencyIdBowAccuracy}, got)

	_, ok = testSkillDefs.Weapon(AttackType{Type: "unknown"})
	assert.False(t, ok, "攻撃の種類の行が無ければ ok=false")
}
