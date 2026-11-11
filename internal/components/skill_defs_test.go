package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

// testSkillDefs は components のテストが使うスキルの定義の表。値は raw の行と同じにする
var testSkillDefs = func() SkillDefs {
	abilities := map[oapi.AbilityId][]oapi.SkillId{
		oapi.Str: {oapi.SkillIdSword, oapi.SkillIdSpear, oapi.SkillIdFist, oapi.SkillIdWeightBearing},
		oapi.Sen: {oapi.SkillIdBow, oapi.SkillIdHandgun, oapi.SkillIdRifle, oapi.SkillIdCannon, oapi.SkillIdExploration},
		oapi.Dex: {oapi.SkillIdCrafting, oapi.SkillIdSmithing, oapi.SkillIdNegotiation, oapi.SkillIdMechanic},
		oapi.Agi: {oapi.SkillIdSprinting, oapi.SkillIdStealth, oapi.SkillIdNightVision},
		oapi.Vit: {oapi.SkillIdColdResist, oapi.SkillIdHungerResist, oapi.SkillIdHealing},
		oapi.Def: {oapi.SkillIdHeavyArmor, oapi.SkillIdFireResist, oapi.SkillIdThunderResist, oapi.SkillIdChillResist, oapi.SkillIdPhotonResist},
	}
	skills := map[oapi.SkillId]SkillDef{}
	for abl, ids := range abilities {
		for _, id := range ids {
			skills[id] = SkillDef{Name: i18n.Msgid(id), Ability: abl}
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
	weapons := []WeaponDef{
		{AttackType: oapi.AttackCategorySword, Name: "Sword", Range: oapi.AttackRangeMelee},
		{AttackType: oapi.AttackCategorySpear, Name: "Spear", Range: oapi.AttackRangeMelee},
		{AttackType: oapi.AttackCategoryFist, Name: "Martial arts", Range: oapi.AttackRangeMelee},
		{AttackType: oapi.AttackCategoryBow, Name: "Bow", Range: oapi.AttackRangeRanged, RangeParams: RangeParams{OptimalRange: 4, MaxRange: 10, PenaltyPerTile: 5}},
		{AttackType: oapi.AttackCategoryHandgun, Name: "Handgun", Range: oapi.AttackRangeRanged, RangeParams: RangeParams{OptimalRange: 3, MaxRange: 8, PenaltyPerTile: 8}},
		{AttackType: oapi.AttackCategoryRifle, Name: "Rifle", Range: oapi.AttackRangeRanged, RangeParams: RangeParams{OptimalRange: 8, MaxRange: 16, PenaltyPerTile: 3}},
		{AttackType: oapi.AttackCategoryCannon, Name: "Cannon", Range: oapi.AttackRangeRanged, RangeParams: RangeParams{OptimalRange: 6, MaxRange: 12, PenaltyPerTile: 5}},
	}
	for i := range weapons {
		w := &weapons[i]
		w.Skill = oapi.SkillId(w.AttackType)
		w.Damage = oapi.ProficiencyId(string(w.AttackType) + "_damage")
		w.Accuracy = oapi.ProficiencyId(string(w.AttackType) + "_accuracy")
		limb := BodyFuncManipulation
		if w.Range == oapi.AttackRangeRanged {
			limb = BodyFuncSight
		}
		proficiencies[w.Damage] = ProficiencyDef{Skill: w.Skill, PerLevel: 5}
		proficiencies[w.Accuracy] = ProficiencyDef{Skill: w.Skill, PerLevel: 3, BodyFunc: limb}
	}
	categories := []SkillCategory{{Name: "Melee", IDs: []oapi.SkillId{oapi.SkillIdSword, oapi.SkillIdSpear, oapi.SkillIdFist}}}
	resists := []ResistDef{
		{Element: oapi.ElementFIRE, Proficiency: oapi.ProficiencyIdFireResist},
		{Element: oapi.ElementTHUNDER, Proficiency: oapi.ProficiencyIdThunderResist},
		{Element: oapi.ElementCHILL, Proficiency: oapi.ProficiencyIdChillResist},
		{Element: oapi.ElementPHOTON, Proficiency: oapi.ProficiencyIdPhotonResist},
	}
	return SkillDefs{Skills: skills, Categories: categories, Proficiencies: proficiencies, Weapons: weapons, Resists: resists}
}()

func TestSkillDefs_Ability(t *testing.T) {
	t.Parallel()

	assert.Equal(t, oapi.Str, testSkillDefs.Ability(oapi.SkillIdSword))
	assert.Equal(t, oapi.Def, testSkillDefs.Ability(oapi.SkillIdFireResist))
	assert.Panics(t, func() { testSkillDefs.Ability("undefined_skill") }, "未登録は panic する")
}

func TestSkillDefs_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, i18n.Msgid("sword"), testSkillDefs.Name(oapi.SkillIdSword))
	assert.Equal(t, i18n.Msgid("undefined_skill"), testSkillDefs.Name("undefined_skill"), "未登録は id をそのまま返す")
}

func TestSkillDefs_Weapon(t *testing.T) {
	t.Parallel()

	got, ok := testSkillDefs.Weapon(oapi.AttackCategoryBow)
	assert.True(t, ok)
	assert.Equal(t, WeaponDef{
		AttackType: oapi.AttackCategoryBow, Name: "Bow", Range: oapi.AttackRangeRanged,
		RangeParams: RangeParams{OptimalRange: 4, MaxRange: 10, PenaltyPerTile: 5},
		Skill:       oapi.SkillIdBow, Damage: oapi.ProficiencyIdBowDamage, Accuracy: oapi.ProficiencyIdBowAccuracy,
	}, got)

	_, ok = testSkillDefs.Weapon(oapi.AttackCategory("unknown"))
	assert.False(t, ok, "攻撃の種類の行が無ければ ok=false")
}

func TestSkillDefs_Resist(t *testing.T) {
	t.Parallel()

	got, ok := testSkillDefs.Resist(oapi.ElementCHILL)
	assert.True(t, ok)
	assert.Equal(t, oapi.ProficiencyIdChillResist, got)

	_, ok = testSkillDefs.Resist(oapi.ElementNONE)
	assert.False(t, ok, "無属性を軽減する倍率は無い")
}

func TestSkillDefs_RangeParams(t *testing.T) {
	t.Parallel()

	got, ok := testSkillDefs.RangeParams(oapi.AttackCategoryRifle)
	assert.True(t, ok)
	assert.Equal(t, RangeParams{OptimalRange: 8, MaxRange: 16, PenaltyPerTile: 3}, got)

	_, ok = testSkillDefs.RangeParams(oapi.AttackCategorySword)
	assert.False(t, ok, "近接は射程を持たない")
}
