package lifecycle_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

// addCubeFuel はキューブ収納に材質と重量を持つ燃料アイテムを1つ足す
func addCubeFuel(t *testing.T, world w.World, cube ecs.Entity, kind oapi.Material, mg consts.Milligram) {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.Material.Add(e, &gc.Material{Kind: kind})
	world.Components.Weight.Add(e, &gc.Weight{Milligram: mg})
	world.Components.LocationInStorage.Add(e, &gc.LocationInStorage{Owner: cube})
}

func TestConsumeCubeFuel_足りれば消費してtrue(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	// COAL 2kg = 1600 の1個。必要量1000は丸ごと1個で賄い、端数はその1個を使い切る
	addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(2*consts.MilligramPerKg))

	assert.True(t, lifecycle.ConsumeCubeFuel(world, cube, consts.Heat(1000)))
	assert.Equal(t, consts.Heat(0), query.CubeFuelTotal(world, cube), "丸ごと消費で残量0")
}

func TestConsumeCubeFuel_足りなければfalseで無消費(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	// COAL 1kg = 800 のみ。必要量1000に届かない
	addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(1*consts.MilligramPerKg))

	assert.False(t, lifecycle.ConsumeCubeFuel(world, cube, consts.Heat(1000)))
	assert.Equal(t, consts.Heat(800), query.CubeFuelTotal(world, cube), "不足時は何も消費しない")
}
