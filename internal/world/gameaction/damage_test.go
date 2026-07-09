package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestApplyHealing(t *testing.T) {
	t.Parallel()

	t.Run("HPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.HP, &gc.HP{Max: 100, Current: 50})
		entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		actual := ApplyHealing(world, entity, 30)
		assert.Equal(t, 30, actual)

		hp := world.Components.HP.MustGet(entity)
		assert.Equal(t, 80, hp.Current)
	})

	t.Run("最大HPを超えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.HP, &gc.HP{Max: 100, Current: 90})
		entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		actual := ApplyHealing(world, entity, 50)
		assert.Equal(t, 10, actual, "実際の回復量は10のみ")

		hp := world.Components.HP.MustGet(entity)
		assert.Equal(t, 100, hp.Current)
	})

	t.Run("HP満タンなら回復量は0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.HP, &gc.HP{Max: 100, Current: 100})
		entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

		actual := ApplyHealing(world, entity, 10)
		assert.Equal(t, 0, actual)
	})
}

func TestReactToHostileAction(t *testing.T) {
	t.Parallel()

	t.Run("CombatIgnoreはCombatAttackに変化する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		solo := &gc.SoloAI{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore}
		ai := &gc.AI{Planner: solo}
		entity.AddComponent(world.Components.AI, ai)

		reactToHostileAction(world, entity)

		assert.Equal(t, gc.CombatAttack, solo.CombatCurrent)
		assert.Equal(t, gc.CombatIgnore, solo.CombatDefault)
	})

	t.Run("CombatEvadeは変化しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		solo := &gc.SoloAI{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade}
		ai := &gc.AI{Planner: solo}
		entity.AddComponent(world.Components.AI, ai)

		reactToHostileAction(world, entity)

		assert.Equal(t, gc.CombatEvade, solo.CombatCurrent)
		assert.Equal(t, gc.CombatEvade, solo.CombatDefault)
	})

	t.Run("CombatAttackは変化しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		solo := &gc.SoloAI{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack}
		ai := &gc.AI{Planner: solo}
		entity.AddComponent(world.Components.AI, ai)

		reactToHostileAction(world, entity)

		assert.Equal(t, gc.CombatAttack, solo.CombatCurrent)
	})

	t.Run("AIがないエンティティではpanicしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()

		assert.NotPanics(t, func() {
			reactToHostileAction(world, entity)
		})
	})
}

func TestApplyDamage_Prop(t *testing.T) {
	t.Parallel()

	t.Run("ダメージでHPが減少する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source := world.Manager.NewEntity()
		source.AddComponent(world.Components.Player, &gc.Player{})

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "木箱"})
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.HP, &gc.HP{Max: 30, Current: 30})

		ApplyDamage(world, prop, 10, source)

		hp := world.Components.HP.MustGet(prop)
		assert.Equal(t, 20, hp.Current)
		assert.False(t, prop.HasComponent(world.Components.Dead))
	})

	t.Run("HPが0になるとDeadが付与される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source := world.Manager.NewEntity()
		source.AddComponent(world.Components.Player, &gc.Player{})

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "木箱"})
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.HP, &gc.HP{Max: 30, Current: 10})

		ApplyDamage(world, prop, 10, source)

		hp := world.Components.HP.MustGet(prop)
		assert.Equal(t, 0, hp.Current)
		assert.True(t, prop.HasComponent(world.Components.Dead))
	})

	t.Run("過剰ダメージでもHPは0で止まる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source := world.Manager.NewEntity()
		source.AddComponent(world.Components.Player, &gc.Player{})

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "木箱"})
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.HP, &gc.HP{Max: 30, Current: 5})

		ApplyDamage(world, prop, 100, source)

		hp := world.Components.HP.MustGet(prop)
		assert.Equal(t, 0, hp.Current)
		assert.True(t, prop.HasComponent(world.Components.Dead))
	})
}
