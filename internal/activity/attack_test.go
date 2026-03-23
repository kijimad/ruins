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

	// StatsChangedフラグが立っている
	assert.True(t, actor.HasComponent(world.Components.StatsChanged), "再計算フラグが立っている")
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

func TestGrowWeaponSkill_RangedWeapon(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Sensation: gc.Ability{Total: 10},
	}
	actor.AddComponent(world.Components.Abilities, abils)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, nil))

	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackRifle}

	aa.growWeaponSkill(actor, world, attack)

	// 小銃スキルに経験値が入る
	assert.Greater(t, skills.Data[gc.SkillRifle].Exp.Current, 0, "小銃スキルに経験値が入る")
	// 他のスキルには影響しない
	assert.Equal(t, 0, skills.Data[gc.SkillSword].Exp.Current, "刀剣スキルは変わらない")
	assert.Equal(t, 0, skills.Data[gc.SkillHandgun].Exp.Current, "拳銃スキルは変わらない")
}

func TestGrowWeaponSkill_OnlyAffectsMatchingSkill(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	actor.AddComponent(world.Components.Abilities, abils)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, nil))

	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackSpear}

	aa.growWeaponSkill(actor, world, attack)

	assert.Greater(t, skills.Data[gc.SkillSpear].Exp.Current, 0, "長物スキルに経験値が入る")
	assert.Equal(t, 0, skills.Data[gc.SkillSword].Exp.Current, "刀剣スキルは変わらない")
	assert.Equal(t, 0, skills.Data[gc.SkillFist].Exp.Current, "格闘スキルは変わらない")
}

func TestGrowWeaponSkill_MaxLevelStopsGrowth(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	skills.Data[gc.SkillSword].Value = 100 // 最大レベル
	skills.Data[gc.SkillSword].Exp.Current = 99
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 10},
	}
	actor.AddComponent(world.Components.Abilities, abils)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, nil))

	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackSword}

	aa.growWeaponSkill(actor, world, attack)

	assert.Equal(t, 100, skills.Data[gc.SkillSword].Value, "最大レベルを超えない")
	assert.Equal(t, 99, skills.Data[gc.SkillSword].Exp.Current, "経験値が変わらない")
}

func TestGrowWeaponSkill_LevelUpWithHealthStatus(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	skills.Data[gc.SkillSword].Exp.Current = 95
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	actor.AddComponent(world.Components.Abilities, abils)

	hs := &gc.HealthStatus{}
	hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
		Type:     gc.ConditionHypothermia,
		Severity: gc.SeverityMinor,
	})
	actor.AddComponent(world.Components.HealthStatus, hs)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, hs))

	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackSword}

	aa.growWeaponSkill(actor, world, attack)

	assert.Equal(t, 1, skills.Data[gc.SkillSword].Value, "スキルアップしている")

	// 再計算されたCharModifiersにHealthStatusのペナルティが反映されている
	mods := world.Components.CharModifiers.Get(actor).(*gc.CharModifiers)
	require.NotNil(t, mods)
	// MoveCost = 100 + 走破Lv0*(-2) + AGI0*(-1) + 軽度低体温10 = 110
	assert.Equal(t, 110, mods.MoveCost, "HealthStatusのペナルティがCharModifiers再計算に反映されている")
}

func TestGrowWeaponSkill_UnknownWeaponType(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	actor.AddComponent(world.Components.Abilities, abils)

	aa := &AttackActivity{}
	attack := &gc.Attack{AttackCategory: gc.AttackType{Type: "unknown"}}

	aa.growWeaponSkill(actor, world, attack)

	// 全スキルの経験値が0のまま
	for _, id := range gc.AllSkillIDs {
		assert.Equal(t, 0, skills.Data[id].Exp.Current, "スキル %s の経験値が変わらない", id)
	}
}
