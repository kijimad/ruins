package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestApplyHealing(t *testing.T) {
	t.Parallel()

	t.Run("HPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Max: 100, Current: 50})
		world.Components.GridElement.Add(entity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})

		actual := ApplyHealing(world, entity, 30)
		assert.Equal(t, 30, actual)

		hp := world.Components.HP.Get(entity)
		assert.Equal(t, 80, hp.Current)
	})

	t.Run("最大HPを超えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Max: 100, Current: 90})
		world.Components.GridElement.Add(entity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})

		actual := ApplyHealing(world, entity, 50)
		assert.Equal(t, 10, actual, "実際の回復量は10のみ")

		hp := world.Components.HP.Get(entity)
		assert.Equal(t, 100, hp.Current)
	})

	t.Run("HP満タンなら回復量は0", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.HP.Add(entity, &gc.HP{Max: 100, Current: 100})
		world.Components.GridElement.Add(entity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}})

		actual := ApplyHealing(world, entity, 10)
		assert.Equal(t, 0, actual)
	})
}

func TestReactToHostileAction(t *testing.T) {
	t.Parallel()

	t.Run("CombatIgnoreはCombatAttackに変化する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.SoloAI.Add(entity, &gc.SoloAI{CombatDefault: gc.CombatIgnore, CombatCurrent: gc.CombatIgnore})

		reactToHostileAction(world, entity)

		solo := world.Components.SoloAI.Get(entity)
		assert.Equal(t, gc.CombatAttack, solo.CombatCurrent)
		assert.Equal(t, gc.CombatIgnore, solo.CombatDefault)
	})

	t.Run("CombatEvadeは変化しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.SoloAI.Add(entity, &gc.SoloAI{CombatDefault: gc.CombatEvade, CombatCurrent: gc.CombatEvade})

		reactToHostileAction(world, entity)

		solo := world.Components.SoloAI.Get(entity)
		assert.Equal(t, gc.CombatEvade, solo.CombatCurrent)
		assert.Equal(t, gc.CombatEvade, solo.CombatDefault)
	})

	t.Run("CombatAttackは変化しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.SoloAI.Add(entity, &gc.SoloAI{CombatDefault: gc.CombatAttack, CombatCurrent: gc.CombatAttack})

		reactToHostileAction(world, entity)

		solo := world.Components.SoloAI.Get(entity)
		assert.Equal(t, gc.CombatAttack, solo.CombatCurrent)
	})

	t.Run("AIがないエンティティではpanicしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()

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

		source := world.ECS.NewEntity()
		world.Components.Player.Add(source, &gc.Player{})

		prop := world.ECS.NewEntity()
		world.Components.Name.Add(prop, &gc.Name{Name: "木箱"})
		world.Components.Prop.Add(prop, &gc.Prop{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 30})

		ApplyDamage(world, prop, 10, source)

		hp := world.Components.HP.Get(prop)
		assert.Equal(t, 20, hp.Current)
		assert.False(t, world.Components.Dead.Has(prop))
	})

	t.Run("HPが0になるとDeadが付与される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source := world.ECS.NewEntity()
		world.Components.Player.Add(source, &gc.Player{})

		prop := world.ECS.NewEntity()
		world.Components.Name.Add(prop, &gc.Name{Name: "木箱"})
		world.Components.Prop.Add(prop, &gc.Prop{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 10})

		ApplyDamage(world, prop, 10, source)

		hp := world.Components.HP.Get(prop)
		assert.Equal(t, 0, hp.Current)
		assert.True(t, world.Components.Dead.Has(prop))
	})

	t.Run("過剰ダメージでもHPは0で止まる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source := world.ECS.NewEntity()
		world.Components.Player.Add(source, &gc.Player{})

		prop := world.ECS.NewEntity()
		world.Components.Name.Add(prop, &gc.Name{Name: "木箱"})
		world.Components.Prop.Add(prop, &gc.Prop{})
		world.Components.HP.Add(prop, &gc.HP{Max: 30, Current: 5})

		ApplyDamage(world, prop, 100, source)

		hp := world.Components.HP.Get(prop)
		assert.Equal(t, 0, hp.Current)
		assert.True(t, world.Components.Dead.Has(prop))
	})
}
