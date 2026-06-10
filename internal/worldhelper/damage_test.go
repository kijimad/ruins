package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestReactToHostileAction(t *testing.T) {
	t.Parallel()

	t.Run("NeutralはHostileに変化する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		disposition := &gc.Disposition{Default: gc.DispositionNeutral, Current: gc.DispositionNeutral}
		entity.AddComponent(world.Components.Disposition, disposition)

		reactToHostileAction(world, entity)

		assert.Equal(t, gc.DispositionHostile, disposition.Current)
		assert.Equal(t, gc.DispositionNeutral, disposition.Default)
	})

	t.Run("CowardlyはFleeingに変化する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		disposition := &gc.Disposition{Default: gc.DispositionCowardly, Current: gc.DispositionCowardly}
		entity.AddComponent(world.Components.Disposition, disposition)

		reactToHostileAction(world, entity)

		assert.Equal(t, gc.DispositionFleeing, disposition.Current)
		assert.Equal(t, gc.DispositionCowardly, disposition.Default)
	})

	t.Run("Hostileは変化しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()
		disposition := &gc.Disposition{Default: gc.DispositionHostile, Current: gc.DispositionHostile}
		entity.AddComponent(world.Components.Disposition, disposition)

		reactToHostileAction(world, entity)

		assert.Equal(t, gc.DispositionHostile, disposition.Current)
	})

	t.Run("Dispositionがないエンティティではpanicしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.Manager.NewEntity()

		assert.NotPanics(t, func() {
			reactToHostileAction(world, entity)
		})
	})
}

func TestApplyDamage_Breakable(t *testing.T) {
	t.Parallel()

	t.Run("ダメージでHPが減少する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		source := world.Manager.NewEntity()
		source.AddComponent(world.Components.Player, &gc.Player{})

		prop := world.Manager.NewEntity()
		prop.AddComponent(world.Components.Name, &gc.Name{Name: "木箱"})
		prop.AddComponent(world.Components.Prop, nil)
		prop.AddComponent(world.Components.HP, &gc.HP{Pool: gc.Pool{Max: 30, Current: 30}})

		ApplyDamage(world, prop, 10, source)

		hp := world.Components.HP.Get(prop).(*gc.HP)
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
		prop.AddComponent(world.Components.HP, &gc.HP{Pool: gc.Pool{Max: 30, Current: 10}})

		ApplyDamage(world, prop, 10, source)

		hp := world.Components.HP.Get(prop).(*gc.HP)
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
		prop.AddComponent(world.Components.HP, &gc.HP{Pool: gc.Pool{Max: 30, Current: 5}})

		ApplyDamage(world, prop, 100, source)

		hp := world.Components.HP.Get(prop).(*gc.HP)
		assert.Equal(t, 0, hp.Current)
		assert.True(t, prop.HasComponent(world.Components.Dead))
	})
}
