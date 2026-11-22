package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

// testSkillDefs は components のテストが使うスキルの定義の表。値は raw の行と同じにする
var testSkillDefs = func() SkillDefs {
	abilities := map[rawschema.AbilityId][]rawschema.SkillId{
		rawschema.Str: {"sword", "spear", "fist", "weight_bearing"},
		rawschema.Sen: {"bow", "handgun", "rifle", "cannon", "exploration"},
		rawschema.Dex: {"crafting", "smithing", "negotiation", "mechanic"},
		rawschema.Agi: {"sprinting", "stealth", "night_vision"},
		rawschema.Vit: {"cold_resist", "hunger_resist", "healing"},
		rawschema.Def: {"heavy_armor", "fire_resist", "thunder_resist", "chill_resist", "photon_resist"},
	}
	skills := map[rawschema.SkillId]SkillDef{}
	for abl, ids := range abilities {
		for _, id := range ids {
			skills[id] = SkillDef{Name: i18n.Msgid(id), Ability: abl}
		}
	}
	proficiencies := map[rawschema.ProficiencyId]ProficiencyDef{
		rawschema.ProficiencyIdFireResist:     {Skill: "fire_resist", PerLevel: -3},
		rawschema.ProficiencyIdThunderResist:  {Skill: "thunder_resist", PerLevel: -3},
		rawschema.ProficiencyIdChillResist:    {Skill: "chill_resist", PerLevel: -3},
		rawschema.ProficiencyIdPhotonResist:   {Skill: "photon_resist", PerLevel: -3},
		rawschema.ProficiencyIdColdProgress:   {Skill: "cold_resist", PerLevel: -3},
		rawschema.ProficiencyIdHungerProgress: {Skill: "hunger_resist", PerLevel: -2},
		rawschema.ProficiencyIdHealingEffect:  {Skill: "healing", PerLevel: 5},
		rawschema.ProficiencyIdMaxWeight:      {Skill: "weight_bearing", PerLevel: 4},
		rawschema.ProficiencyIdExploration:    {Skill: "exploration", PerLevel: 4},
		rawschema.ProficiencyIdEnemyVision:    {Skill: "stealth", PerLevel: -3},
		rawschema.ProficiencyIdNightVision:    {Skill: "night_vision", PerLevel: 5},
		rawschema.ProficiencyIdMoveCost:       {Skill: "sprinting", PerLevel: -2},
		rawschema.ProficiencyIdCraftCost:      {Skill: "crafting", PerLevel: -3},
		rawschema.ProficiencyIdSmithQuality:   {Skill: "smithing", PerLevel: 3},
		rawschema.ProficiencyIdBuyPrice:       {Skill: "negotiation", PerLevel: -2},
		rawschema.ProficiencyIdSellPrice:      {Skill: "negotiation", PerLevel: 2},
		rawschema.ProficiencyIdHeavyArmor:     {Skill: "heavy_armor", PerLevel: -5},
	}
	weapons := []WeaponDef{
		{AttackType: rawschema.AttackCategorySword, Name: "Sword", Range: rawschema.AttackRangeMelee},
		{AttackType: rawschema.AttackCategorySpear, Name: "Spear", Range: rawschema.AttackRangeMelee},
		{AttackType: rawschema.AttackCategoryFist, Name: "Martial arts", Range: rawschema.AttackRangeMelee},
		{AttackType: rawschema.AttackCategoryBow, Name: "Bow", Range: rawschema.AttackRangeRanged, RangeParams: RangeParams{OptimalRange: 4, MaxRange: 10, PenaltyPerTile: 5}},
		{AttackType: rawschema.AttackCategoryHandgun, Name: "Handgun", Range: rawschema.AttackRangeRanged, RangeParams: RangeParams{OptimalRange: 3, MaxRange: 8, PenaltyPerTile: 8}},
		{AttackType: rawschema.AttackCategoryRifle, Name: "Rifle", Range: rawschema.AttackRangeRanged, RangeParams: RangeParams{OptimalRange: 8, MaxRange: 16, PenaltyPerTile: 3}},
		{AttackType: rawschema.AttackCategoryCannon, Name: "Cannon", Range: rawschema.AttackRangeRanged, RangeParams: RangeParams{OptimalRange: 6, MaxRange: 12, PenaltyPerTile: 5}},
	}
	for i := range weapons {
		w := &weapons[i]
		w.Skill = rawschema.SkillId(w.AttackType)
		w.Damage = rawschema.ProficiencyId(string(w.AttackType) + "_damage")
		w.Accuracy = rawschema.ProficiencyId(string(w.AttackType) + "_accuracy")
		limb := BodyFuncManipulation
		if w.Range == rawschema.AttackRangeRanged {
			limb = BodyFuncSight
		}
		proficiencies[w.Damage] = ProficiencyDef{Skill: w.Skill, PerLevel: 5}
		proficiencies[w.Accuracy] = ProficiencyDef{Skill: w.Skill, PerLevel: 3, BodyFunc: limb}
	}
	categories := []SkillCategory{{Name: "Melee", IDs: []rawschema.SkillId{"sword", "spear", "fist"}}}
	resists := []ResistDef{
		{Element: rawschema.ElementFIRE, Proficiency: rawschema.ProficiencyIdFireResist},
		{Element: rawschema.ElementTHUNDER, Proficiency: rawschema.ProficiencyIdThunderResist},
		{Element: rawschema.ElementCHILL, Proficiency: rawschema.ProficiencyIdChillResist},
		{Element: rawschema.ElementPHOTON, Proficiency: rawschema.ProficiencyIdPhotonResist},
	}
	return SkillDefs{Skills: skills, Categories: categories, Proficiencies: proficiencies, Weapons: weapons, Resists: resists}
}()

func TestSkillDefs_Ability(t *testing.T) {
	t.Parallel()

	assert.Equal(t, rawschema.Str, testSkillDefs.Ability("sword"))
	assert.Equal(t, rawschema.Def, testSkillDefs.Ability("fire_resist"))
	assert.Panics(t, func() { testSkillDefs.Ability("undefined_skill") }, "未登録は panic する")
}

func TestSkillDefs_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, i18n.Msgid("sword"), testSkillDefs.Name("sword"))
	assert.Equal(t, i18n.Msgid("undefined_skill"), testSkillDefs.Name("undefined_skill"), "未登録は id をそのまま返す")
}

func TestSkillDefs_Weapon(t *testing.T) {
	t.Parallel()

	got, ok := testSkillDefs.Weapon(rawschema.AttackCategoryBow)
	assert.True(t, ok)
	assert.Equal(t, WeaponDef{
		AttackType: rawschema.AttackCategoryBow, Name: "Bow", Range: rawschema.AttackRangeRanged,
		RangeParams: RangeParams{OptimalRange: 4, MaxRange: 10, PenaltyPerTile: 5},
		Skill:       "bow", Damage: rawschema.ProficiencyIdBowDamage, Accuracy: rawschema.ProficiencyIdBowAccuracy,
	}, got)

	_, ok = testSkillDefs.Weapon(rawschema.AttackCategory("unknown"))
	assert.False(t, ok, "攻撃の種類の行が無ければ ok=false")
}

func TestSkillDefs_Resist(t *testing.T) {
	t.Parallel()

	got, ok := testSkillDefs.Resist(rawschema.ElementCHILL)
	assert.True(t, ok)
	assert.Equal(t, rawschema.ProficiencyIdChillResist, got)

	_, ok = testSkillDefs.Resist(rawschema.ElementNONE)
	assert.False(t, ok, "無属性を軽減する倍率は無い")
}

func TestSkillDefs_RangeParams(t *testing.T) {
	t.Parallel()

	got, ok := testSkillDefs.RangeParams(rawschema.AttackCategoryRifle)
	assert.True(t, ok)
	assert.Equal(t, RangeParams{OptimalRange: 8, MaxRange: 16, PenaltyPerTile: 3}, got)

	_, ok = testSkillDefs.RangeParams(rawschema.AttackCategorySword)
	assert.False(t, ok, "近接は射程を持たない")
}
