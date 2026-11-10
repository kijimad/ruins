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
		oapi.ProficiencyIdFireResist:     {oapi.SkillIdFireResist, -3},
		oapi.ProficiencyIdThunderResist:  {oapi.SkillIdThunderResist, -3},
		oapi.ProficiencyIdChillResist:    {oapi.SkillIdChillResist, -3},
		oapi.ProficiencyIdPhotonResist:   {oapi.SkillIdPhotonResist, -3},
		oapi.ProficiencyIdColdProgress:   {oapi.SkillIdColdResist, -3},
		oapi.ProficiencyIdHungerProgress: {oapi.SkillIdHungerResist, -2},
		oapi.ProficiencyIdHealingEffect:  {oapi.SkillIdHealing, 5},
		oapi.ProficiencyIdMaxWeight:      {oapi.SkillIdWeightBearing, 4},
		oapi.ProficiencyIdExploration:    {oapi.SkillIdExploration, 4},
		oapi.ProficiencyIdEnemyVision:    {oapi.SkillIdStealth, -3},
		oapi.ProficiencyIdNightVision:    {oapi.SkillIdNightVision, 5},
		oapi.ProficiencyIdMoveCost:       {oapi.SkillIdSprinting, -2},
		oapi.ProficiencyIdCraftCost:      {oapi.SkillIdCrafting, -3},
		oapi.ProficiencyIdSmithQuality:   {oapi.SkillIdSmithing, 3},
		oapi.ProficiencyIdBuyPrice:       {oapi.SkillIdNegotiation, -2},
		oapi.ProficiencyIdSellPrice:      {oapi.SkillIdNegotiation, 2},
		oapi.ProficiencyIdHeavyArmor:     {oapi.SkillIdHeavyArmor, -5},
	}
	for _, id := range WeaponSkillIDs {
		proficiencies[WeaponDamageKey(id)] = ProficiencyDef{id, 5}
		proficiencies[WeaponAccuracyKey(id)] = ProficiencyDef{id, 3}
	}
	categories := []SkillCategory{{Name: "Melee", IDs: []oapi.SkillId{oapi.SkillIdSword, oapi.SkillIdSpear, oapi.SkillIdFist}}}
	return SkillDefs{Skills: skills, Categories: categories, Proficiencies: proficiencies}
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
