package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadActivity_Validate_NoTarget(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: nil}
	assert.Error(t, ra.Validate(comp, actor, world))
}

func TestReadActivity_Validate_NotABook(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	item := world.ECS.NewEntity()

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &item}
	assert.Error(t, ra.Validate(comp, actor, world))
}

func TestReadActivity_Validate_AlreadyCompleted(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	bookEntity := world.ECS.NewEntity()

	book := &gc.Book{
		Effort: gc.IntPool{Max: 10, Current: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 1},
	}
	world.Components.Book.Add(bookEntity, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &bookEntity}
	err := ra.Validate(comp, actor, world)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "読了済み")
}

func TestReadActivity_Validate_RequiredLevelNotMet(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})
	world.Components.Skills.Add(actor, gc.NewSkills())

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 5, RequiredLevel: 3},
	}
	world.Components.Book.Add(bookEntity, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &bookEntity}
	err := ra.Validate(comp, actor, world)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "レベル3以上必要")
}

func TestReadActivity_Validate_RequiredLevelMet(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})

	skills := gc.NewSkills()
	skills.Get(gc.SkillSword).Value = 3
	world.Components.Skills.Add(actor, skills)

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 5, RequiredLevel: 3},
	}
	world.Components.Book.Add(bookEntity, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &bookEntity}
	assert.NoError(t, ra.Validate(comp, actor, world))
}

func TestReadActivity_Validate_Success(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 1},
	}
	world.Components.Book.Add(bookEntity, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &bookEntity}
	assert.NoError(t, ra.Validate(comp, actor, world))
}

func TestReadActivity_DoTurn_AdvancesProgress(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})
	world.Components.Skills.Add(actor, gc.NewSkills())
	world.Components.Abilities.Add(actor, &gc.Abilities{})

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 100},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	world.Components.Book.Add(bookEntity, book)
	book = world.Components.Book.Get(bookEntity)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   100,
		TurnsLeft:    100,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	require.NoError(t, err)
	assert.Equal(t, 10, book.Effort.Current, "基本工数10ぶん進んでいる")
	assert.Equal(t, 99, comp.TurnsLeft, "ターンが1減っている")
}

func TestReadActivity_DoTurn_GainsSkillExp(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})

	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)
	world.Components.Abilities.Add(actor, &gc.Abilities{Strength: gc.Ability{Total: 5}})
	world.Components.CharModifiers.Add(actor, gc.RecalculateCharModifiers(skills, nil, nil))

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	world.Components.Book.Add(bookEntity, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	before := skills.Get(gc.SkillSword).Exp.Current
	err := ra.DoTurn(comp, actor, world)
	require.NoError(t, err)
	assert.Greater(t, skills.Get(gc.SkillSword).Exp.Current, before, "経験値が増加する")
}

func TestReadActivity_DoTurn_NoExpWhenTooHard(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})

	skills := gc.NewSkills()
	world.Components.Skills.Add(actor, skills)
	world.Components.Abilities.Add(actor, &gc.Abilities{})

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 10}, // プレイヤーLv0, 本Lv10 → diff=10 → 0%
	}
	world.Components.Book.Add(bookEntity, book)
	book = world.Components.Book.Get(bookEntity)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	require.NoError(t, err)
	assert.Equal(t, 0, skills.Get(gc.SkillSword).Exp.Current, "難しすぎて経験値を得られない")
	assert.Equal(t, 10, book.Effort.Current, "工数は進む")
}

func TestReadActivity_DoTurn_CompletesWhenEffortReached(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})
	world.Components.Skills.Add(actor, gc.NewSkills())
	world.Components.Abilities.Add(actor, &gc.Abilities{})

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 15, Current: 8},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	world.Components.Book.Add(bookEntity, book)
	book = world.Components.Book.Get(bookEntity)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   5,
		TurnsLeft:    1,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	require.NoError(t, err)
	assert.True(t, book.IsCompleted(), "読了している")
	assert.Equal(t, gc.ActivityStateCompleted, comp.State, "アクティビティが完了している")
}

func TestReadActivity_DoTurn_CanceledByEnemy(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.Player.Add(actor, &gc.Player{})
	world.Components.FactionAlly.Add(actor, &gc.FactionAlly{})
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})

	// 隣に敵を配置
	enemy := world.ECS.NewEntity()
	world.Components.FactionEnemy.Add(enemy, &gc.FactionEnemy{})
	world.Components.GridElement.Add(enemy, &gc.GridElement{X: 6, Y: 5})

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	world.Components.Book.Add(bookEntity, book)
	book = world.Components.Book.Get(bookEntity)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	require.NoError(t, err)
	assert.Equal(t, gc.ActivityStateCanceled, comp.State, "敵がいるのでキャンセルされる")
	assert.Equal(t, 0, book.Effort.Current, "章は進んでいない")
}

func TestReadActivity_DoTurn_SkillLevelUp(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})
	world.Components.Player.Add(actor, &gc.Player{})

	skills := gc.NewSkills()
	skills.Get(gc.SkillSword).Exp.Current = 95 // スキルアップ直前
	world.Components.Skills.Add(actor, skills)

	abils := &gc.Abilities{Strength: gc.Ability{Total: 5}}
	world.Components.Abilities.Add(actor, abils)
	world.Components.CharModifiers.Add(actor, gc.RecalculateCharModifiers(skills, abils, nil))

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	world.Components.Book.Add(bookEntity, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	require.NoError(t, err)

	assert.Equal(t, 1, skills.Get(gc.SkillSword).Value, "スキルアップしている")

	// StatsChangedフラグが立っている
	assert.True(t, world.Components.StatsChanged.Has(actor), "再計算フラグが立っている")
}

func TestReadActivity_NoSkillsComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.ECS.NewEntity()
	world.Components.GridElement.Add(actor, &gc.GridElement{X: 5, Y: 5})
	// Skillsコンポーネントなし → panicしない

	bookEntity := world.ECS.NewEntity()
	book := &gc.Book{
		Effort: gc.IntPool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	world.Components.Book.Add(bookEntity, book)
	book = world.Components.Book.Get(bookEntity)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	require.NoError(t, err)
	assert.Equal(t, 10, book.Effort.Current, "工数は進む")
}
