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
