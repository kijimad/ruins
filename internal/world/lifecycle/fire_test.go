package lifecycle_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
)

func TestAddFuel(t *testing.T) {
	t.Parallel()

	t.Run("燃えている火にくべると残ターンが増え燃料が消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 0})

		fuel := world.ECS.NewEntity()
		world.Components.Material.Add(fuel, &gc.Material{Kind: oapi.WOOD})
		world.Components.Weight.Add(fuel, &gc.Weight{Milligram: 200 * consts.MilligramPerGram})

		lifecycle.AddFuel(world, fire, fuel)

		// WOOD 200/kg × 200g = 40。地面直の効率50%で 40*50/100 = 20 ターン
		assert.Equal(t, consts.Turn(20), world.Components.Burning.Get(fire).Remaining, "残ターンへ畳み込む")
		assert.False(t, world.ECS.Alive(fuel), "くべた燃料は消費される")
	})

	t.Run("燃えていない火にはくべず燃料も消費しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Burning を持たない火。メニュー表示中に燃え尽きた状況に相当する
		fire := world.ECS.NewEntity()

		fuel := world.ECS.NewEntity()
		world.Components.Material.Add(fuel, &gc.Material{Kind: oapi.WOOD})
		world.Components.Weight.Add(fuel, &gc.Weight{Milligram: 200 * consts.MilligramPerGram})

		lifecycle.AddFuel(world, fire, fuel)

		assert.True(t, world.ECS.Alive(fuel), "火が燃えていなければ燃料は消費されない")
	})
}
