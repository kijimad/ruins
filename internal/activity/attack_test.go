package activity

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSkillMult_NoSkills(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	// Skillsコンポーネントなし → 100を返す
	melee := &gc.Melee{AttackCategory: gc.AttackSword}
	assert.Equal(t, 100, int(getSkillMult(entity, melee, world, true)))
	assert.Equal(t, 100, int(getSkillMult(entity, melee, world, false)))
}

func TestGetSkillMult_NilAttack(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()
	assert.Equal(t, 100, int(getSkillMult(entity, nil, world, true)))
}

func TestGetSkillMult_WithSkills(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()

	skills := gc.NewSkills()
	skills.Get(oapi.SkillIdSword).Value = 3
	world.Components.Skills.Add(entity, skills)

	melee := &gc.Melee{AttackCategory: gc.AttackSword}
	// 刀剣Lv3: ダメージ倍率 = 100 + 3*5 = 115
	assert.Equal(t, 115, int(getSkillMult(entity, melee, world, true)))
	// 刀剣Lv3: 命中倍率 = 100 + 3*3 = 109
	assert.Equal(t, 109, int(getSkillMult(entity, melee, world, false)))
}

func TestGetSkillMult_UnmappedWeapon(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()

	world.Components.Skills.Add(entity, gc.NewSkills())

	// 未登録の武器種 → 100
	melee := &gc.Melee{AttackCategory: gc.AttackType{Type: "unknown"}}
	assert.Equal(t, 100, int(getSkillMult(entity, melee, world, true)))
}

func TestElementMult(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fireResist  int
		hasSkills   bool
		element     oapi.Element
		wantPercent consts.Percent
	}{
		{"スキルを持たない対象は等倍", 0, false, oapi.ElementFIRE, 100},
		{"無属性は耐性があっても等倍", 40, true, oapi.ElementNONE, 100},
		{"空の属性は等倍", 40, true, oapi.Element(""), 100},
		{"耐火Lv5は100+5*(-3)で85", 5, true, oapi.ElementFIRE, 85},
		{"耐火Lv40は100+40*(-3)で-20", 40, true, oapi.ElementFIRE, -20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			target := world.ECS.NewEntity()
			if tt.hasSkills {
				skills := gc.NewSkills()
				skills.Get(oapi.SkillIdFireResist).Value = tt.fireResist
				world.Components.Skills.Add(target, skills)
			}

			assert.Equal(t, tt.wantPercent, elementMult(target, tt.element, world))
		})
	}
}

func TestGrowWeaponSkill_NoSkillsComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	// Skillsコンポーネントなし → panicしない

	melee := &gc.Melee{AttackCategory: gc.AttackSword}
	growWeaponSkill(actor, world, melee)
}

func TestGrowWeaponSkill_NilAttack(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	growWeaponSkill(actor, world, nil)
}

func TestGrowWeaponSkill_GainsExp(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 0},
	}
	world.Components.Abilities.Add(actor, abils)

	melee := &gc.Melee{AttackCategory: gc.AttackSword}

	before := skills.Get(oapi.SkillIdSword).Exp.Current
	growWeaponSkill(actor, world, melee)
	after := skills.Get(oapi.SkillIdSword).Exp.Current

	assert.Greater(t, after, before, "経験値が増加する")
}

func TestGrowWeaponSkill_LevelUpRecalculates(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	skills := gc.NewSkills()
	// スキルアップ直前まで経験値を溜める
	skills.Get(oapi.SkillIdSword).Exp.Current = 95
	world.Components.Skills.Add(actor, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	world.Components.Abilities.Add(actor, abils)

	melee := &gc.Melee{AttackCategory: gc.AttackSword}

	growWeaponSkill(actor, world, melee)

	assert.Equal(t, 1, skills.Get(oapi.SkillIdSword).Value, "スキルアップしている")
}

func TestGrowWeaponSkill_NoAbilitiesComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)
	// Abilitiesコンポーネントなし → スキップされpanicしない

	melee := &gc.Melee{AttackCategory: gc.AttackSword}
	growWeaponSkill(actor, world, melee)

	// 経験値は増えていない（Abilitiesがないので早期リターン）
	assert.Equal(t, 0, skills.Get(oapi.SkillIdSword).Exp.Current)
}

func TestGrowWeaponSkill_Fire(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)

	abils := &gc.Abilities{
		Sensation: gc.Ability{Total: 10},
	}
	world.Components.Abilities.Add(actor, abils)

	fire := &gc.Fire{AttackCategory: gc.AttackRifle}

	growWeaponSkill(actor, world, fire)

	// 小銃スキルに経験値が入る
	assert.Positive(t, skills.Get(oapi.SkillIdRifle).Exp.Current, "小銃スキルに経験値が入る")
	// 他のスキルには影響しない
	assert.Equal(t, 0, skills.Get(oapi.SkillIdSword).Exp.Current, "刀剣スキルは変わらない")
	assert.Equal(t, 0, skills.Get(oapi.SkillIdHandgun).Exp.Current, "拳銃スキルは変わらない")
}

func TestGrowWeaponSkill_敵は成長しない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.FactionEnemy.Add(actor, &gc.FactionEnemy{})

	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)
	abils := &gc.Abilities{Sensation: gc.Ability{Total: 10}}
	world.Components.Abilities.Add(actor, abils)

	growWeaponSkill(actor, world, &gc.Fire{AttackCategory: gc.AttackRifle})

	// 敵は Skills を持っても戦闘中に成長しない。スキル成長は味方側の進行要素
	assert.Equal(t, 0, skills.Get(oapi.SkillIdRifle).Exp.Current, "敵はスキル成長しない")
}

func TestGrowWeaponSkill_OnlyAffectsMatchingSkill(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	world.Components.Abilities.Add(actor, abils)

	melee := &gc.Melee{AttackCategory: gc.AttackSpear}

	growWeaponSkill(actor, world, melee)

	assert.Positive(t, skills.Get(oapi.SkillIdSpear).Exp.Current, "長物スキルに経験値が入る")
	assert.Equal(t, 0, skills.Get(oapi.SkillIdSword).Exp.Current, "刀剣スキルは変わらない")
	assert.Equal(t, 0, skills.Get(oapi.SkillIdFist).Exp.Current, "格闘スキルは変わらない")
}

func TestGrowWeaponSkill_MaxLevelStopsGrowth(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	skills := gc.NewSkills()
	skills.Get(oapi.SkillIdSword).Value = 100 // 最大レベル
	skills.Get(oapi.SkillIdSword).Exp.Current = 99
	world.Components.Skills.Add(actor, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 10},
	}
	world.Components.Abilities.Add(actor, abils)

	melee := &gc.Melee{AttackCategory: gc.AttackSword}

	growWeaponSkill(actor, world, melee)

	assert.Equal(t, 100, skills.Get(oapi.SkillIdSword).Value, "最大レベルを超えない")
	assert.Equal(t, 99, skills.Get(oapi.SkillIdSword).Exp.Current, "経験値が変わらない")
}

func TestGrowWeaponSkill_LevelUpWithHealthStatus(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	skills := gc.NewSkills()
	skills.Get(oapi.SkillIdSword).Exp.Current = 95
	world.Components.Skills.Add(actor, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	world.Components.Abilities.Add(actor, abils)

	hs := &gc.HealthStatus{}
	hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{
		Type:     gc.ConditionType("hypothermia"),
		Severity: gc.SeverityMinor,
	})
	world.Components.HealthStatus.Add(actor, hs)

	melee := &gc.Melee{AttackCategory: gc.AttackSword}

	growWeaponSkill(actor, world, melee)

	assert.Equal(t, 1, skills.Get(oapi.SkillIdSword).Value, "スキルアップしている")

	// 低体温は MoveCost でなく身体機能へ効く。軽度の全身性: 意識=100-10-6/2=87、歩行=100*87/100=87。
	// 意識は master 乗数として歩行へ掛かる
	bodyFuncs := world.Components.HealthStatus.Get(actor).BodyFuncs(world.Resources.ConditionDefs)
	assert.Equal(t, 87, int(bodyFuncs.Consciousness), "HealthStatusが意識へ反映される")
	assert.Equal(t, 87, int(bodyFuncs.Moving), "意識が歩行へ乗算される")
}

func TestApplyAttackDamage_InterruptsActivity(t *testing.T) {
	t.Parallel()

	t.Run("中断可能なアクティビティは被ダメージでキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// 攻撃者の Dexterity 99 で命中率がほぼ上限。固定シードで命中を確定させ、被ダメージでのキャンセルを無分岐で検証する
		world.Resources.Config.RNG = rand.New(rand.NewPCG(7, 0))

		target := world.ECS.NewEntity()
		world.Components.HP.Add(target, &gc.HP{Current: 100, Max: 100})
		world.Components.Abilities.Add(target, &gc.Abilities{
			Agility: gc.Ability{Total: 0},
		})

		// 中断可能なアクティビティ（休息）を設定
		comp := NewActivity(gc.BehaviorRest, 10)
		comp.State = gc.ActivityStateRunning
		world.Components.Activity.Add(target, comp)

		// 攻撃者をセットアップ（命中率を最大にするため高い器用度）
		attacker := world.ECS.NewEntity()
		world.Components.Abilities.Add(attacker, &gc.Abilities{
			Strength:  gc.Ability{Total: 50},
			Dexterity: gc.Ability{Total: 99},
		})

		melee := &gc.Melee{
			Damage:         100,
			AttackCategory: gc.AttackFist,
		}

		require.NoError(t, applyAttackDamage(attacker, target, world, melee, "テスト攻撃", 0, 0))

		// CancelActivity は RemoveActivity で Activity を削除するので、キャンセルは Has で観測する
		assert.False(t, world.Components.Activity.Has(target),
			"被ダメージで中断可能なアクティビティがキャンセルされる")
	})

	t.Run("中断不可のアクティビティは被ダメージでもキャンセルされない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		target := world.ECS.NewEntity()
		world.Components.HP.Add(target, &gc.HP{Current: 10000, Max: 10000})
		world.Components.Abilities.Add(target, &gc.Abilities{
			Agility: gc.Ability{Total: 0},
		})

		// 中断不可のアクティビティ（攻撃）を設定
		comp := NewActivity(gc.BehaviorMelee, 0)
		comp.State = gc.ActivityStateRunning
		world.Components.Activity.Add(target, comp)

		attacker := world.ECS.NewEntity()
		world.Components.Abilities.Add(attacker, &gc.Abilities{
			Strength:  gc.Ability{Total: 50},
			Dexterity: gc.Ability{Total: 99},
		})

		melee := &gc.Melee{
			Damage:         1,
			AttackCategory: gc.AttackFist,
		}

		require.NoError(t, applyAttackDamage(attacker, target, world, melee, "テスト攻撃", 0, 0))

		// 中断不可なので生存中はアクティビティが残る
		assert.True(t, world.Components.Activity.Has(target),
			"中断不可のアクティビティは被ダメージでも残る")
		activityComp := world.Components.Activity.Get(target)
		assert.Equal(t, gc.ActivityStateRunning, activityComp.State)
	})

	t.Run("死亡時は中断不可のアクティビティもキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		target := world.ECS.NewEntity()
		world.Components.HP.Add(target, &gc.HP{Current: 1, Max: 100})
		world.Components.Abilities.Add(target, &gc.Abilities{
			Agility: gc.Ability{Total: 0},
		})

		// 中断不可のアクティビティ（攻撃）を設定
		comp := NewActivity(gc.BehaviorMelee, 0)
		comp.State = gc.ActivityStateRunning
		world.Components.Activity.Add(target, comp)

		attacker := world.ECS.NewEntity()
		world.Components.Abilities.Add(attacker, &gc.Abilities{
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
		if world.Components.Dead.Has(target) {
			assert.True(t, world.Components.Activity.Has(target),
				"applyAttackDamage時点ではアクティビティは削除されない")
		}
	})
}

func TestGetAttackParams_プレイヤーが装備した近接武器のパラメータを返す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	weapon, err := lifecycle.SpawnBackpackItem(world, "torch", 1)
	require.NoError(t, err)
	lifecycle.MoveToEquip(world, weapon, player, gc.SlotWeapon1)
	query.GetWeaponSelection(world).Slot = 1

	attack, name, err := getAttackParams(player, world)
	require.NoError(t, err)
	assert.Equal(t, "Torch", name)
	melee, ok := attack.(*gc.Melee)
	require.True(t, ok)
	// 具体値は torch の raw 定義依存なので契約だけを見る。装備武器のパラメータが返る
	assert.Positive(t, melee.Damage)
}

func TestGetAttackParams_プレイヤーが武器未装備なら素手にフォールバックする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	// プレイヤーは生成時に松明を武器スロット1へ装備している。外して未装備の状態を作る
	require.NoError(t, lifecycle.UnequipAll(world, player))
	query.GetWeaponSelection(world).Slot = 1

	attack, name, err := getAttackParams(player, world)
	require.NoError(t, err)
	assert.Equal(t, "Bare Hands", name)
	melee, ok := attack.(*gc.Melee)
	require.True(t, ok)
	// 素手の Damage 値は getBareHandsAttack のテストで固定する。ここは素手へ落ちる契約だけを見る
	assert.Positive(t, melee.Damage)
}

func TestGetAttackParams_不正なスロット番号でエラーになる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	// 武器スロットは1-5が正当範囲
	query.GetWeaponSelection(world).Slot = 6

	_, _, err = getAttackParams(player, world)
	require.ErrorIs(t, err, ErrInvalidWeaponSlot)
}

func TestGetAttackParams_敵はCommandTableから攻撃を取得する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "bat")
	require.NoError(t, err)

	attack, name, err := getAttackParams(enemy, world)
	require.NoError(t, err)
	// 具体的な攻撃 id は bat の CommandTable 定義依存なので、攻撃が取れて Melee である契約だけを見る
	assert.NotEmpty(t, name)
	_, ok := attack.(*gc.Melee)
	require.True(t, ok)
}

func TestGetAttackParams_CommandTable取得失敗で素手にフォールバックする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	enemy := world.ECS.NewEntity()
	// 未登録のコマンドテーブル名 → GetAttackFromCommandTableが失敗する
	world.Components.CommandTable.Add(enemy, &gc.CommandTable{Name: "no_such_table"})

	attack, name, err := getAttackParams(enemy, world)
	require.NoError(t, err)
	assert.Equal(t, "Bare Hands", name)
	melee, ok := attack.(*gc.Melee)
	require.True(t, ok)
	// 素手の Damage 値は getBareHandsAttack のテストで固定する。ここは素手へ落ちる契約だけを見る
	assert.Positive(t, melee.Damage)
}

func TestGetAttackParams_プレイヤーでも敵でもない場合はエラーになる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	entity := world.ECS.NewEntity()

	_, _, err := getAttackParams(entity, world)
	require.ErrorIs(t, err, ErrNoAttackSource)
}

func TestGetBareHandsAttack_素手武器のパラメータを返す(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	attack, name, err := getBareHandsAttack(world)
	require.NoError(t, err)
	assert.Equal(t, "Bare Hands", name)
	assert.Equal(t, 3, attack.GetDamage())
}

func TestMeleeBehavior_DoTurn(t *testing.T) {
	t.Parallel()

	t.Run("射程内の対象への攻撃が成功して完了する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "bat")
		require.NoError(t, err)
		// applyAttackDamageの命中判定がRNGに依存するため、固定シードで命中を保証する
		world.Resources.Config.RNG = rand.New(rand.NewPCG(7, 0))

		comp := NewMeleeActivity(enemy)
		comp.State = gc.ActivityStateRunning

		ab := &MeleeBehavior{}
		err = ab.DoTurn(comp, player, world)

		require.NoError(t, err)
		assert.Equal(t, gc.ActivityStateCompleted, comp.State)
	})

	t.Run("Paramsの型が不一致だとキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		comp := &gc.Activity{
			BehaviorName: gc.BehaviorMelee,
			State:        gc.ActivityStateRunning,
			Params:       &gc.MoveParams{},
		}

		ab := &MeleeBehavior{}
		err = ab.DoTurn(comp, player, world)

		require.ErrorIs(t, err, ErrParamsTypeMismatch)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})

	t.Run("射程外の対象だとキャンセルされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 20, Y: 20}, "bat")
		require.NoError(t, err)

		comp := NewMeleeActivity(enemy)
		comp.State = gc.ActivityStateRunning

		ab := &MeleeBehavior{}
		err = ab.DoTurn(comp, player, world)

		require.ErrorIs(t, err, ErrAttackTargetInvalid)
		assert.Equal(t, gc.ActivityStateCanceled, comp.State)
	})
}

func TestMeleeBehavior_canAttack(t *testing.T) {
	t.Parallel()

	t.Run("Paramsの型が不一致ならfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)

		comp := &gc.Activity{Params: &gc.MoveParams{}}
		ab := &MeleeBehavior{}

		assert.False(t, ab.canAttack(comp, player, world))
	})

	t.Run("Validateが通ればtrue", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
		require.NoError(t, err)
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, "bat")
		require.NoError(t, err)

		comp := NewMeleeActivity(enemy)
		ab := &MeleeBehavior{}

		assert.True(t, ab.canAttack(comp, player, world))
	})
}

func TestGrowWeaponSkill_UnknownWeaponType(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)

	abils := &gc.Abilities{
		Strength: gc.Ability{Total: 5},
	}
	world.Components.Abilities.Add(actor, abils)

	melee := &gc.Melee{AttackCategory: gc.AttackType{Type: "unknown"}}

	growWeaponSkill(actor, world, melee)

	assert.Equal(t, map[oapi.SkillId]*gc.Skill{}, world.Components.Skills.Get(actor).Data, "どのスキルにも経験値が入らない")
}

func TestApplyAttackDamage_道具が必須のprop(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		weapon     string
		wantDamage bool
	}{
		{"斧で殴ると削れる", "axe", true},
		{"松明で殴っても削れずスキルも伸びない", "torch", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
			require.NoError(t, err)
			require.NoError(t, lifecycle.UnequipAll(world, player))
			weapon, err := lifecycle.SpawnBackpackItem(world, tt.weapon, 1)
			require.NoError(t, err)
			lifecycle.MoveToEquip(world, weapon, player, gc.SlotWeapon1)
			query.GetWeaponSelection(world).Slot = 1
			tree, err := lifecycle.SpawnProp(world, "tree_a", 10, 9)
			require.NoError(t, err)
			attack, name, err := getAttackParams(player, world)
			require.NoError(t, err)
			w, ok := world.Resources.SkillDefs.Weapon(attack.GetAttackCategory())
			require.True(t, ok)
			skillID := w.Skill
			expBefore := world.Components.Skills.Get(player).Get(skillID).Exp.Current

			// 命中率は上限があり外れうるので、補正を大きく取って何度も殴り、少なくとも1回は当てる
			for range 20 {
				require.NoError(t, applyAttackDamage(player, tree, world, attack, name, 1000, 0))
			}

			hp := world.Components.HP.Get(tree)
			expAfter := world.Components.Skills.Get(player).Get(skillID).Exp.Current
			if tt.wantDamage {
				assert.Less(t, hp.Current, hp.Max)
			} else {
				assert.Equal(t, hp.Max, hp.Current)
				assert.Equal(t, expBefore, expAfter)
			}
		})
	}
}
