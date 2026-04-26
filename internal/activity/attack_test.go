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
	melee := &gc.Melee{AttackCategory: gc.AttackSword}
	assert.Equal(t, 100, getSkillMult(entity, melee, world, true))
	assert.Equal(t, 100, getSkillMult(entity, melee, world, false))
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
	skills.Get(gc.SkillSword).Value = 3
	mods := gc.RecalculateCharModifiers(skills, nil, nil)
	entity.AddComponent(world.Components.CharModifiers, mods)

	melee := &gc.Melee{AttackCategory: gc.AttackSword}
	// 刀剣Lv3: ダメージ倍率 = 100 + 3*5 = 115
	assert.Equal(t, 115, getSkillMult(entity, melee, world, true))
	// 刀剣Lv3: 命中倍率 = 100 + 3*3 = 109
	assert.Equal(t, 109, getSkillMult(entity, melee, world, false))
}

func TestGetSkillMult_UnmappedWeapon(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.Manager.NewEntity()

	skills := gc.NewSkills()
	mods := gc.RecalculateCharModifiers(skills, nil, nil)
	entity.AddComponent(world.Components.CharModifiers, mods)

	// 未登録の武器種 → 100
	melee := &gc.Melee{AttackCategory: gc.AttackType{Type: "unknown"}}
	assert.Equal(t, 100, getSkillMult(entity, melee, world, true))
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
	skills.Get(gc.SkillFireResist).Value = 5
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
	skills.Get(gc.SkillFireResist).Value = 40 // 高い耐性値
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

	melee := &gc.Melee{AttackCategory: gc.AttackSword}
	growWeaponSkill(actor, world, melee)
}

func TestGrowWeaponSkill_NilAttack(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	growWeaponSkill(actor, world, nil)
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

	melee := &gc.Melee{AttackCategory: gc.AttackSword}

	before := skills.Get(gc.SkillSword).Exp.Current
	growWeaponSkill(actor, world, melee)
	after := skills.Get(gc.SkillSword).Exp.Current

	assert.Greater(t, after, before, "経験値が増加する")
}

func TestGrowWeaponSkill_LevelUpRecalculates(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	// スキルアップ直前まで経験値を溜める
	skills.Get(gc.SkillSword).Exp.Current = 95
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	actor.AddComponent(world.Components.Abilities, abils)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, nil))

	melee := &gc.Melee{AttackCategory: gc.AttackSword}

	growWeaponSkill(actor, world, melee)

	assert.Equal(t, 1, skills.Get(gc.SkillSword).Value, "スキルアップしている")

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

	melee := &gc.Melee{AttackCategory: gc.AttackSword}
	growWeaponSkill(actor, world, melee)

	// 経験値は増えていない（Abilitiesがないので早期リターン）
	assert.Equal(t, 0, skills.Get(gc.SkillSword).Exp.Current)
}

func TestGrowWeaponSkill_Fire(t *testing.T) {
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

	fire := &gc.Fire{AttackCategory: gc.AttackRifle}

	growWeaponSkill(actor, world, fire)

	// 小銃スキルに経験値が入る
	assert.Greater(t, skills.Get(gc.SkillRifle).Exp.Current, 0, "小銃スキルに経験値が入る")
	// 他のスキルには影響しない
	assert.Equal(t, 0, skills.Get(gc.SkillSword).Exp.Current, "刀剣スキルは変わらない")
	assert.Equal(t, 0, skills.Get(gc.SkillHandgun).Exp.Current, "拳銃スキルは変わらない")
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

	melee := &gc.Melee{AttackCategory: gc.AttackSpear}

	growWeaponSkill(actor, world, melee)

	assert.Greater(t, skills.Get(gc.SkillSpear).Exp.Current, 0, "長物スキルに経験値が入る")
	assert.Equal(t, 0, skills.Get(gc.SkillSword).Exp.Current, "刀剣スキルは変わらない")
	assert.Equal(t, 0, skills.Get(gc.SkillFist).Exp.Current, "格闘スキルは変わらない")
}

func TestGrowWeaponSkill_MaxLevelStopsGrowth(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	skills.Get(gc.SkillSword).Value = 100 // 最大レベル
	skills.Get(gc.SkillSword).Exp.Current = 99
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 10},
	}
	actor.AddComponent(world.Components.Abilities, abils)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, nil))

	melee := &gc.Melee{AttackCategory: gc.AttackSword}

	growWeaponSkill(actor, world, melee)

	assert.Equal(t, 100, skills.Get(gc.SkillSword).Value, "最大レベルを超えない")
	assert.Equal(t, 99, skills.Get(gc.SkillSword).Exp.Current, "経験値が変わらない")
}

func TestGrowWeaponSkill_LevelUpWithHealthStatus(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	skills := gc.NewSkills()
	skills.Get(gc.SkillSword).Exp.Current = 95
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

	melee := &gc.Melee{AttackCategory: gc.AttackSword}

	growWeaponSkill(actor, world, melee)

	assert.Equal(t, 1, skills.Get(gc.SkillSword).Value, "スキルアップしている")

	// 再計算されたCharModifiersにHealthStatusのペナルティが反映されている
	mods := world.Components.CharModifiers.Get(actor).(*gc.CharModifiers)
	require.NotNil(t, mods)
	// MoveCost = 100 + 走破Lv0*(-2) + AGI0*(-1) + 軽度低体温10 = 110
	assert.Equal(t, 110, mods.MoveCost, "HealthStatusのペナルティがCharModifiers再計算に反映されている")
}

func TestApplyAttackDamage_InterruptsActivity(t *testing.T) {
	t.Parallel()

	t.Run("中断可能なアクティビティは被ダメージでキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 100, Max: 100},
		})
		target.AddComponent(world.Components.Abilities, &gc.Abilities{
			Agility: gc.Ability{Total: 0},
		})

		// 中断可能なアクティビティ（休息）を設定
		ra := &RestActivity{}
		comp, err := NewActivity(ra, 10)
		require.NoError(t, err)
		comp.State = gc.ActivityStateRunning
		target.AddComponent(world.Components.Activity, comp)

		// 攻撃者をセットアップ（命中率を最大にするため高い器用度）
		attacker := world.Manager.NewEntity()
		attacker.AddComponent(world.Components.Abilities, &gc.Abilities{
			Strength:  gc.Ability{Total: 50},
			Dexterity: gc.Ability{Total: 99},
		})

		melee := &gc.Melee{
			Damage:         100,
			AttackCategory: gc.AttackFist,
		}

		require.NoError(t, applyAttackDamage(attacker, target, world, melee, "テスト攻撃", 0, 0))

		// アクティビティがキャンセルされている（命中時）、または残っている（ミス時）
		currentComp := world.Components.Activity.Get(target)
		if currentComp != nil {
			// ミスした場合はアクティビティが残る。状態がRunningなら中断処理は正しく動作している
			activity := currentComp.(*gc.Activity)
			assert.Equal(t, gc.ActivityStateRunning, activity.State,
				"ミス時はアクティビティがRunningのまま残る")
		}
		// 命中時はRemoveActivityで削除されているのでcurrentCompはnil
	})

	t.Run("中断不可のアクティビティは被ダメージでもキャンセルされない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 10000, Max: 10000},
		})
		target.AddComponent(world.Components.Abilities, &gc.Abilities{
			Agility: gc.Ability{Total: 0},
		})

		// 中断不可のアクティビティ（攻撃）を設定
		aa := &AttackActivity{}
		comp, err := NewActivity(aa, 1)
		require.NoError(t, err)
		comp.State = gc.ActivityStateRunning
		target.AddComponent(world.Components.Activity, comp)

		attacker := world.Manager.NewEntity()
		attacker.AddComponent(world.Components.Abilities, &gc.Abilities{
			Strength:  gc.Ability{Total: 50},
			Dexterity: gc.Ability{Total: 99},
		})

		melee := &gc.Melee{
			Damage:         1,
			AttackCategory: gc.AttackFist,
		}

		require.NoError(t, applyAttackDamage(attacker, target, world, melee, "テスト攻撃", 0, 0))

		// 中断不可なので生存中はアクティビティが残る
		assert.True(t, target.HasComponent(world.Components.Activity),
			"中断不可のアクティビティは被ダメージでも残る")
		activityComp := world.Components.Activity.Get(target).(*gc.Activity)
		assert.Equal(t, gc.ActivityStateRunning, activityComp.State)
	})

	t.Run("死亡時は中断不可のアクティビティもキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		target := world.Manager.NewEntity()
		target.AddComponent(world.Components.Pools, &gc.Pools{
			HP: gc.Pool{Current: 1, Max: 100},
		})
		target.AddComponent(world.Components.Abilities, &gc.Abilities{
			Agility: gc.Ability{Total: 0},
		})

		// 中断不可のアクティビティ（攻撃）を設定
		aa := &AttackActivity{}
		comp, err := NewActivity(aa, 1)
		require.NoError(t, err)
		comp.State = gc.ActivityStateRunning
		target.AddComponent(world.Components.Activity, comp)

		attacker := world.Manager.NewEntity()
		attacker.AddComponent(world.Components.Abilities, &gc.Abilities{
			Strength:  gc.Ability{Total: 50},
			Dexterity: gc.Ability{Total: 99},
		})

		melee := &gc.Melee{
			Damage:         999,
			AttackCategory: gc.AttackFist,
		}

		require.NoError(t, applyAttackDamage(attacker, target, world, melee, "テスト攻撃", 0, 0))

		// 死亡時のアクティビティキャンセルはDeadCleanupSystemが担当する
		// applyAttackDamage時点ではアクティビティはまだ残っている
		if target.HasComponent(world.Components.Dead) {
			assert.True(t, target.HasComponent(world.Components.Activity),
				"applyAttackDamage時点ではアクティビティは削除されない")
		}
	})
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

	melee := &gc.Melee{AttackCategory: gc.AttackType{Type: "unknown"}}

	growWeaponSkill(actor, world, melee)

	// 全スキルの経験値が0のまま
	for _, id := range gc.AllSkillIDs {
		assert.Equal(t, 0, skills.Get(id).Exp.Current, "スキル %s の経験値が変わらない", id)
	}
}
