package raw

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewItemSpec_複数のオプション要素が設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "回復薬"
Description = "傷薬"
SpriteSheetName = "field"
SpriteKey = "potion"
Value = 100
Weight = "200 g"
Stackable = true
ProvidesNutrition = 10
InflictsDamage = 5

[Items.Consumable]
TargetGroup = "ALLY"
TargetNum = "SINGLE"
UsableScene = "ANY"

[Items.ProvidesHealing]
Amount = 0
Ratio = 0.5
ValueType = "PERCENTAGE"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewItemSpec(raws, "回復薬")
	require.NoError(t, err)

	require.NotNil(t, spec.Consumable)
	assert.Equal(t, gc.TargetGroupType("ALLY"), spec.Consumable.TargetType.TargetGroup)
	assert.Equal(t, gc.TargetNumType("SINGLE"), spec.Consumable.TargetType.TargetNum)

	require.NotNil(t, spec.ProvidesHealing)
	assert.Equal(t, gc.HealRatio, spec.ProvidesHealing.Kind)
	assert.InDelta(t, 0.5, spec.ProvidesHealing.Amount, 0.0001)

	require.NotNil(t, spec.ProvidesNutrition)
	assert.Equal(t, 10, spec.ProvidesNutrition.Amount)

	require.NotNil(t, spec.InflictsDamage)
	assert.Equal(t, 5, spec.InflictsDamage.Amount)

	require.NotNil(t, spec.Weight)
	assert.Equal(t, int64(200000), int64(spec.Weight.Milligram))
}

func TestNewItemSpec_回復が数値型で設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "包帯"
Description = "止血する"

[Items.ProvidesHealing]
Amount = 15
Ratio = 0
ValueType = "NUMERAL"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewItemSpec(raws, "包帯")
	require.NoError(t, err)

	require.NotNil(t, spec.ProvidesHealing)
	assert.Equal(t, gc.HealNumeral, spec.ProvidesHealing.Kind)
	assert.InDelta(t, 15.0, spec.ProvidesHealing.Amount, 0.0001)
}

func TestNewItemSpec_弾薬が設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "9mm弾"
Description = "拳銃弾"

[Items.Ammo]
AccuracyBonus = 5
AmmoTag = "9mm"
DamageBonus = -2
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewItemSpec(raws, "9mm弾")
	require.NoError(t, err)

	require.NotNil(t, spec.Ammo)
	assert.Equal(t, "9mm", spec.Ammo.AmmoTag)
	assert.Equal(t, -2, spec.Ammo.DamageBonus)
	assert.Equal(t, 5, spec.Ammo.AccuracyBonus)
}

func TestNewItemSpec_装備品と装備ボーナスが設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "革靴"
Description = "軽い靴"

[Items.Wearable]
Defense = 1
EquipmentCategory = "FEET"
InsulationCold = 2
InsulationHeat = 2

[Items.EquipBonus]
Agility = 1
Dexterity = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewItemSpec(raws, "革靴")
	require.NoError(t, err)

	require.NotNil(t, spec.Wearable)
	assert.Equal(t, 1, spec.Wearable.Defense)
	assert.Equal(t, gc.EquipmentType("FEET"), spec.Wearable.EquipmentCategory)
	assert.Equal(t, 1, spec.Wearable.EquipBonus.Agility)
	assert.Equal(t, 1, spec.Wearable.EquipBonus.Dexterity)
}

func TestNewItemSpec_本が設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "探索の本"
Description = "探索術が学べる"

[Items.Book]
TotalEffort = 150

[Items.Book.Skill]
MaxLevel = 2
RequiredLevel = 0
TargetSkill = "exploration"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewItemSpec(raws, "探索の本")
	require.NoError(t, err)

	require.NotNil(t, spec.Book)
	assert.Equal(t, 150, spec.Book.Effort.Max)
	require.NotNil(t, spec.Book.Skill)
	assert.Equal(t, gc.SkillID("exploration"), spec.Book.Skill.TargetSkill)
	assert.Equal(t, 2, spec.Book.Skill.MaxLevel)
}

func TestNewItemSpec_本にスキル未指定はエラー(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "白紙の本"
Description = "何も書かれていない"

[Items.Book]
TotalEffort = 10
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewItemSpec(raws, "白紙の本")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Skillの指定が必要です")
}

func TestNewItemSpec_本に未定義スキルはエラー(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "謎の本"
Description = "未知のスキルを教える"

[Items.Book]
TotalEffort = 10

[Items.Book.Skill]
MaxLevel = 1
RequiredLevel = 0
TargetSkill = "unknown_skill"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewItemSpec(raws, "謎の本")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未定義のスキルID")
}

func TestNewItemSpec_素材が設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "鉄"
Description = "頑丈な金属"
Material = true
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewItemSpec(raws, "鉄")
	require.NoError(t, err)
	assert.NotNil(t, spec.Material)
}

func TestNewItemSpec_不正な重量はエラー(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "不正な重量アイテム"
Description = "重量表記がおかしい"
Weight = "abc"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewItemSpec(raws, "不正な重量アイテム")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "アイテム '不正な重量アイテム' の重量")
}

func TestNewRecipeSpec_レシピが設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Items]]
Name = "レイガン"
Description = "光線を放つ武器"
Value = 500

[Items.Melee]
Damage = 20
Accuracy = 80
AttackCount = 1
Element = "none"
AttackCategory = "SWORD"

[[Recipes]]
Name = "レイガン"

[[Recipes.Inputs]]
Name = "鉄"
Amount = 4

[[Recipes.Inputs]]
Name = "フェライトコア"
Amount = 2
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewRecipeSpec(raws, "レイガン")
	require.NoError(t, err)

	assert.Equal(t, "レイガン", spec.Name.Name)
	require.NotNil(t, spec.Recipe)
	require.Len(t, spec.Recipe.Inputs, 2)
	assert.Equal(t, "鉄", spec.Recipe.Inputs[0].Name)
	assert.Equal(t, 4, spec.Recipe.Inputs[0].Amount)
	assert.Equal(t, "光線を放つ武器", spec.Description.Description)
	require.NotNil(t, spec.Melee)
	assert.Equal(t, 20, spec.Melee.Damage)
}

func TestNewRecipeSpec_レシピ未存在はエラー(t *testing.T) {
	t.Parallel()

	raws, err := DecodeRaws("")
	require.NoError(t, err)

	_, err = NewRecipeSpec(raws, "存在しないレシピ")
	require.Error(t, err)
}

func TestNewRecipeSpec_対応アイテム無しはエラー(t *testing.T) {
	t.Parallel()

	str := `
[[Recipes]]
Name = "対応アイテムなしレシピ"

[[Recipes.Inputs]]
Name = "鉄"
Amount = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewRecipeSpec(raws, "対応アイテムなしレシピ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate item for recipe")
}

func TestNewMemberSpec_不正な派閥はエラー(t *testing.T) {
	t.Parallel()

	str := `
[[Members]]
Name = "不正派閥"
FactionType = "FactionInvalid"
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewMemberSpec(raws, "不正派閥")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "無効な派閥タイプ")
}

func TestNewMemberSpec_不正な戦闘方針はエラー(t *testing.T) {
	t.Parallel()

	str := `
[[Members]]
Name = "不正戦闘方針"
CombatPolicy = "invalid_policy"
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewMemberSpec(raws, "不正戦闘方針")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "無効な戦闘ポリシー")
}

func TestNewMemberSpec_光源が設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Members]]
Name = "光る敵"
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2

[Members.LightSource]
Enabled = true
Radius = 4

[Members.LightSource.Color]
R = 255
G = 200
B = 50
A = 255
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewMemberSpec(raws, "光る敵")
	require.NoError(t, err)

	require.NotNil(t, spec.LightSource)
	assert.True(t, spec.LightSource.Enabled)
	assert.Equal(t, 4, int(spec.LightSource.Radius))
}

func TestNewMemberSpec_会話が設定される(t *testing.T) {
	t.Parallel()

	str := `
[[Members]]
Name = "話す村人"
[Members.Abilities]
Vitality = 10
Strength = 5
Sensation = 3
Dexterity = 3
Agility = 3
Defense = 2

[Members.Dialog]
MessageKey = "villager_greeting"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	spec, err := NewMemberSpec(raws, "話す村人")
	require.NoError(t, err)

	require.NotNil(t, spec.Dialog)
	assert.Equal(t, "villager_greeting", spec.Dialog.MessageKey)
	require.NotNil(t, spec.Interactable)
	assert.Contains(t, spec.Interactable.Interactions, gc.InteractionTalk)
}

func TestGetProfession_職業を取得する(t *testing.T) {
	t.Parallel()

	str := `
[[Professions]]
Id = "hunter"
Name = "猟師"
Description = "野外活動に長けた生存者"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	prof, err := GetProfession(raws, "hunter")
	require.NoError(t, err)
	assert.Equal(t, "hunter", prof.Id)
	assert.Equal(t, "猟師", prof.Name)
	assert.Equal(t, "野外活動に長けた生存者", prof.Description)
}

func TestGetProfession_未存在はエラー(t *testing.T) {
	t.Parallel()

	raws, err := DecodeRaws("")
	require.NoError(t, err)

	_, err = GetProfession(raws, "存在しない職業")
	require.Error(t, err)
}

func TestGetItemGroup_未存在はエラー(t *testing.T) {
	t.Parallel()

	raws, err := DecodeRaws("")
	require.NoError(t, err)

	_, err = GetItemGroup(raws, "存在しないグループ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "アイテムグループが存在しない")
}

func TestGetProp_未存在はエラー(t *testing.T) {
	t.Parallel()

	raws, err := DecodeRaws("")
	require.NoError(t, err)

	_, err = GetProp(raws, "存在しないProp")
	require.Error(t, err)
}

func TestLoadFromFile_存在しないパスはエラー(t *testing.T) {
	t.Parallel()

	_, err := LoadFromFile("metadata/entities/raw/存在しない.toml")
	require.Error(t, err)
}
