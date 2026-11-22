package lifecycle_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddFuel(t *testing.T) {
	t.Parallel()

	t.Run("燃えている火にくべると残ターンが増え燃料が消費される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		fire := world.ECS.NewEntity()
		world.Components.Burning.Add(fire, &gc.Burning{Remaining: 0})

		fuel := world.ECS.NewEntity()
		world.Components.Material.Add(fuel, &gc.Material{Kind: rawschema.WOOD})
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
		world.Components.Material.Add(fuel, &gc.Material{Kind: rawschema.WOOD})
		world.Components.Weight.Add(fuel, &gc.Weight{Milligram: 200 * consts.MilligramPerGram})

		lifecycle.AddFuel(world, fire, fuel)

		assert.True(t, world.ECS.Alive(fuel), "火が燃えていなければ燃料は消費されない")
	})
}

func TestAddFuel_バックパックの松明をくべると持ち主に灯りを写し直させる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	torch, err := lifecycle.SpawnBackpackItem(world, "torch", 1)
	require.NoError(t, err)
	fire := world.ECS.NewEntity()
	world.Components.Burning.Add(fire, &gc.Burning{})
	world.Components.GridElement.Add(fire, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 6, Y: 5}})
	world.Components.StatsChanged.Remove(player)

	lifecycle.AddFuel(world, fire, torch)

	assert.False(t, world.ECS.Alive(torch))
	assert.True(t, world.Components.StatsChanged.Has(player), "くべた松明の灯りを外す")
}
