package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestReadActivity_Validate_NoTarget(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: nil}
	assert.Error(t, ra.Validate(comp, actor, world))
}

func TestReadActivity_Validate_NotABook(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	item := world.Manager.NewEntity()

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &item}
	assert.Error(t, ra.Validate(comp, actor, world))
}

func TestReadActivity_Validate_AlreadyCompleted(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	bookEntity := world.Manager.NewEntity()

	book := &gc.Book{
		Effort: gc.Pool{Max: 10, Current: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 1},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &bookEntity}
	err := ra.Validate(comp, actor, world)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "読了済み")
}

func TestReadActivity_Validate_RequiredLevelNotMet(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	actor.AddComponent(world.Components.Skills, gc.NewSkills())

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 5, RequiredLevel: 3},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &bookEntity}
	err := ra.Validate(comp, actor, world)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "レベル3以上必要")
}

func TestReadActivity_Validate_RequiredLevelMet(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	skills := gc.NewSkills()
	skills.Get(gc.SkillSword).Value = 3
	actor.AddComponent(world.Components.Skills, skills)

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 5, RequiredLevel: 3},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &bookEntity}
	assert.NoError(t, ra.Validate(comp, actor, world))
}

func TestReadActivity_Validate_Success(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 1},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{Target: &bookEntity}
	assert.NoError(t, ra.Validate(comp, actor, world))
}

func TestReadActivity_DoTurn_AdvancesProgress(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	actor.AddComponent(world.Components.Skills, gc.NewSkills())
	actor.AddComponent(world.Components.Abilities, &gc.Abilities{})

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 100},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   100,
		TurnsLeft:    100,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	assert.NoError(t, err)
	assert.Equal(t, 10, book.Effort.Current, "基本工数10ぶん進んでいる")
	assert.Equal(t, 99, comp.TurnsLeft, "ターンが1減っている")
}

func TestReadActivity_DoTurn_GainsSkillExp(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	skills := gc.NewSkills()
	actor.AddComponent(world.Components.Skills, skills)
	actor.AddComponent(world.Components.Abilities, &gc.Abilities{Strength: gc.Ability{Total: 5}})
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, nil, nil))

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	bookEntity.AddComponent(world.Components.Book, book)

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
	assert.NoError(t, err)
	assert.Greater(t, skills.Get(gc.SkillSword).Exp.Current, before, "経験値が増加する")
}

func TestReadActivity_DoTurn_NoExpWhenTooHard(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	skills := gc.NewSkills()
	actor.AddComponent(world.Components.Skills, skills)
	actor.AddComponent(world.Components.Abilities, &gc.Abilities{})

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 10}, // プレイヤーLv0, 本Lv10 → diff=10 → 0%
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	assert.NoError(t, err)
	assert.Equal(t, 0, skills.Get(gc.SkillSword).Exp.Current, "難しすぎて経験値を得られない")
	assert.Equal(t, 10, book.Effort.Current, "工数は進む")
}

func TestReadActivity_DoTurn_CompletesWhenEffortReached(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	actor.AddComponent(world.Components.Skills, gc.NewSkills())
	actor.AddComponent(world.Components.Abilities, &gc.Abilities{})

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 15, Current: 8},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   5,
		TurnsLeft:    1,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	assert.NoError(t, err)
	assert.True(t, book.IsCompleted(), "読了している")
	assert.Equal(t, gc.ActivityStateCompleted, comp.State, "アクティビティが完了している")
}

func TestReadActivity_DoTurn_CanceledByEnemy(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	// 隣に敵を配置
	enemy := world.Manager.NewEntity()
	enemy.AddComponent(world.Components.FactionEnemy, nil)
	enemy.AddComponent(world.Components.GridElement, &gc.GridElement{X: 6, Y: 5})

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	assert.NoError(t, err)
	assert.Equal(t, gc.ActivityStateCanceled, comp.State, "敵がいるのでキャンセルされる")
	assert.Equal(t, 0, book.Effort.Current, "章は進んでいない")
}

func TestReadActivity_DoTurn_SkillLevelUp(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	actor.AddComponent(world.Components.Player, nil)

	skills := gc.NewSkills()
	skills.Get(gc.SkillSword).Exp.Current = 95 // スキルアップ直前
	actor.AddComponent(world.Components.Skills, skills)

	abils := &gc.Abilities{Strength: gc.Ability{Total: 5}}
	actor.AddComponent(world.Components.Abilities, abils)
	actor.AddComponent(world.Components.CharModifiers, gc.RecalculateCharModifiers(skills, abils, nil))

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	assert.NoError(t, err)

	assert.Equal(t, 1, skills.Get(gc.SkillSword).Value, "スキルアップしている")

	// StatsChangedフラグが立っている
	assert.True(t, actor.HasComponent(world.Components.StatsChanged), "再計算フラグが立っている")
}

func TestReadActivity_NoSkillsComponent(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	actor := world.Manager.NewEntity()
	actor.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})
	// Skillsコンポーネントなし → panicしない

	bookEntity := world.Manager.NewEntity()
	book := &gc.Book{
		Effort: gc.Pool{Max: 10},
		Skill:  &gc.SkillBookEffect{TargetSkill: gc.SkillSword, MaxLevel: 0},
	}
	bookEntity.AddComponent(world.Components.Book, book)

	ra := &ReadActivity{}
	comp := &gc.Activity{
		BehaviorName: gc.BehaviorRead,
		State:        gc.ActivityStateRunning,
		TurnsTotal:   10,
		TurnsLeft:    10,
		Target:       &bookEntity,
	}

	err := ra.DoTurn(comp, actor, world)
	assert.NoError(t, err)
	assert.Equal(t, 10, book.Effort.Current, "工数は進む")
}
