package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSkillMult_NoCharModifiers(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.Manager.NewEntity()
	// CharModifiersコンポーネントなし → 100を返す
	attack := &gc.Attack{AttackCategory: gc.AttackSword}
	assert.Equal(t, 100, getSkillMult(entity, attack, world, true))
	assert.Equal(t, 100, getSkillMult(entity, attack, world, false))
}

func TestGetSkillMult_NilAttack(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.Manager.NewEntity()
	assert.Equal(t, 100, getSkillMult(entity, nil, world, true))
}

func TestGetSkillMult_WithCharModifiers(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.Manager.NewEntity()

	skills := gc.NewSkills()
	skills.Data[gc.SkillSword].Value = 3
	mods := gc.RecalculateCharModifiers(skills, nil, nil)
	entity.AddComponent(world.Components.CharModifiers, mods)

	attack := &gc.Attack{AttackCategory: gc.AttackSword}
	// 刀剣Lv3: ダメージ倍率 = 100 + 3*5 = 115
	assert.Equal(t, 115, getSkillMult(entity, attack, world, true))
	// 刀剣Lv3: 命中倍率 = 100 + 3*3 = 109
	assert.Equal(t, 109, getSkillMult(entity, attack, world, false))
}

func TestGetSkillMult_UnmappedWeapon(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.Manager.NewEntity()

	skills := gc.NewSkills()
	mods := gc.RecalculateCharModifiers(skills, nil, nil)
	entity.AddComponent(world.Components.CharModifiers, mods)

	// 未登録の武器種 → 100
	attack := &gc.Attack{AttackCategory: gc.AttackType{Type: "unknown"}}
	assert.Equal(t, 100, getSkillMult(entity, attack, world, true))
}

func TestApplyElementResist_NoCharModifiers(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	target := world.Manager.NewEntity()

	// CharModifiersなし → ダメージそのまま
	assert.Equal(t, 50, applyElementResist(50, target, gc.ElementTypeFire, world))
}

func TestApplyElementResist_WithResist(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	target := world.Manager.NewEntity()

	skills := gc.NewSkills()
	skills.Data[gc.SkillFireResist].Value = 5
	mods := gc.RecalculateCharModifiers(skills, nil, nil)
	target.AddComponent(world.Components.CharModifiers, mods)

	// 耐火Lv5: 耐性 = 100 + 5*(-3) = 85
	// 50 * 85 / 100 = 42
	assert.Equal(t, 42, applyElementResist(50, target, gc.ElementTypeFire, world))
}

func TestApplyElementResist_MinimumDamage(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	target := world.Manager.NewEntity()

	skills := gc.NewSkills()
	skills.Data[gc.SkillFireResist].Value = 40 // 高い耐性値
	mods := gc.RecalculateCharModifiers(skills, nil, nil)
	target.AddComponent(world.Components.CharModifiers, mods)

	// 耐火Lv40: 耐性 = 100 + 40*(-3) = -20
	// ダメージ = 5 * -20 / 100 = -1 → 最低保証1
	assert.Equal(t, MinDamage, applyElementResist(5, target, gc.ElementTypeFire, world))
}

func TestGrowWeaponSkill_NoSkillsComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	// Skillsコンポーネントなし → panicしない
	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackSword}
	aa.growWeaponSkill(actor, world, attack)
}

func TestGrowWeaponSkill_NilAttack(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	aa := &AttackActivity{}
	aa.growWeaponSkill(actor, world, nil)
}

func TestGrowWeaponSkill_GainsExp(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 0},
	}
	actor.AddComponent(world.Components.Abilities, abils)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, nil))

	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackSword}

	before := skills.Data[gc.SkillSword].Exp.Current
	aa.growWeaponSkill(actor, world, attack)
	after := skills.Data[gc.SkillSword].Exp.Current

	assert.Greater(t, after, before, "経験値が増加する")
}

func TestGrowWeaponSkill_LevelUpRecalculates(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	// スキルアップ直前まで経験値を溜める
	skills.Data[gc.SkillSword].Exp.Current = 95
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	actor.AddComponent(world.Components.Abilities, abils)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, nil))

	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackSword}

	aa.growWeaponSkill(actor, world, attack)

	assert.Equal(t, 1, skills.Data[gc.SkillSword].Value, "スキルアップしている")

	// CharModifiersが再計算されている
	mods := world.Components.CharModifiers.Get(actor).(*gc.CharModifiers)
	require.NotNil(t, mods)
	// 刀剣Lv1 + STR5: ダメージ = 100 + 1*5 + 5*1 = 110
	assert.Equal(t, 110, mods.WeaponDamage[gc.SkillSword], "CharModifiersが再計算されている")
}

func TestGrowWeaponSkill_NoAbilitiesComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	actor.AddComponent(world.Components.Skills, skills)
	// Abilitiesコンポーネントなし → スキップされpanicしない

	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackSword}
	aa.growWeaponSkill(actor, world, attack)

	// 経験値は増えていない（Abilitiesがないので早期リターン）
	assert.Equal(t, 0, skills.Data[gc.SkillSword].Exp.Current)
}
