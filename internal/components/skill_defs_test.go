package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSkillDefs は components のテストが使うスキルの定義の表。値は raw の行と同じにする
var testSkillDefs = func() SkillDefs {
	abilities := map[AbilityID][]SkillID{
		AblSTR: {SkillSword, SkillSpear, SkillFist, SkillWeightBearing},
		AblSEN: {SkillBow, SkillHandgun, SkillRifle, SkillCannon, SkillExploration},
		AblDEX: {SkillCrafting, SkillSmithing, SkillNegotiation, SkillMechanic},
		AblAGI: {SkillSprinting, SkillStealth, SkillNightVision},
		AblVIT: {SkillColdResist, SkillHungerResist, SkillHealing},
		AblDEF: {SkillHeavyArmor, SkillFireResist, SkillThunderResist, SkillChillResist, SkillPhotonResist},
	}
	skills := map[SkillID]SkillDef{}
	for abl, ids := range abilities {
		for _, id := range ids {
			skills[id] = SkillDef{Name: string(id), Ability: abl}
		}
	}
	proficiencies := map[ProficiencyKey]ProficiencyDef{
		ProfFireResist:     {SkillFireResist, -3},
		ProfThunderResist:  {SkillThunderResist, -3},
		ProfChillResist:    {SkillChillResist, -3},
		ProfPhotonResist:   {SkillPhotonResist, -3},
		ProfColdProgress:   {SkillColdResist, -3},
		ProfHungerProgress: {SkillHungerResist, -2},
		ProfHealingEffect:  {SkillHealing, 5},
		ProfMaxWeight:      {SkillWeightBearing, 4},
		ProfExploration:    {SkillExploration, 4},
		ProfEnemyVision:    {SkillStealth, -3},
		ProfNightVision:    {SkillNightVision, 5},
		ProfMoveCost:       {SkillSprinting, -2},
		ProfCraftCost:      {SkillCrafting, -3},
		ProfSmithQuality:   {SkillSmithing, 3},
		ProfBuyPrice:       {SkillNegotiation, -2},
		ProfSellPrice:      {SkillNegotiation, 2},
		ProfHeavyArmor:     {SkillHeavyArmor, -5},
	}
	for _, id := range WeaponSkillIDs {
		proficiencies[WeaponDamageKey(id)] = ProficiencyDef{id, 5}
		proficiencies[WeaponAccuracyKey(id)] = ProficiencyDef{id, 3}
	}
	categories := []SkillCategory{{Name: "Melee", IDs: []SkillID{SkillSword, SkillSpear, SkillFist}}}
	return NewSkillDefs(skills, categories, proficiencies)
}()

func TestSkillDefs_Ability(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AblSTR, testSkillDefs.Ability(SkillSword))
	assert.Equal(t, AblDEF, testSkillDefs.Ability(SkillFireResist))
	assert.Panics(t, func() { testSkillDefs.Ability("undefined_skill") }, "未登録は panic する")
}

func TestSkillDefs_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "sword", testSkillDefs.Name(SkillSword))
	assert.Equal(t, "undefined_skill", testSkillDefs.Name("undefined_skill"), "未登録は id をそのまま返す")
}

func TestSkillDefs_Def(t *testing.T) {
	t.Parallel()

	def, ok := testSkillDefs.Def(SkillBow)
	require.True(t, ok)
	assert.Equal(t, SkillDef{Name: "bow", Ability: AblSEN}, def)

	_, ok = testSkillDefs.Def("undefined_skill")
	assert.False(t, ok)
}

func TestSkillDefs_Categories(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []SkillCategory{{Name: "Melee", IDs: []SkillID{SkillSword, SkillSpear, SkillFist}}}, testSkillDefs.Categories())
}
