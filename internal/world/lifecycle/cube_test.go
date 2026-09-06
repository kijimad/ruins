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

// addCubeFuel はキューブ収納に材質と重量を持つ燃料アイテムを1つ足し、その entity を返す
func addCubeFuel(t *testing.T, world w.World, cube ecs.Entity, kind oapi.Material, mg consts.Milligram) ecs.Entity {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.Material.Add(e, &gc.Material{Kind: kind})
	world.Components.Weight.Add(e, &gc.Weight{Milligram: mg})
	world.Components.LocationInStorage.Add(e, &gc.LocationInStorage{Owner: cube})
	return e
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

func TestConsumeCubeFuel_複数アイテムをまたいで消費し必要量で止まる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	// COAL 1kg = 800 を3個。必要量1000は1個で足りず2個目まで消費して止まる。3個目は残す
	a := addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(1*consts.MilligramPerKg))
	b := addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(1*consts.MilligramPerKg))
	c := addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(1*consts.MilligramPerKg))

	assert.True(t, lifecycle.ConsumeCubeFuel(world, cube, consts.Heat(1000)))
	assert.False(t, world.ECS.Alive(a), "1個目は消費される")
	assert.False(t, world.ECS.Alive(b), "2個目も消費して必要量に達する")
	assert.True(t, world.ECS.Alive(c), "必要量に達したら3個目は残す")
	assert.Equal(t, consts.Heat(800), query.CubeFuelTotal(world, cube), "残量は3個目の800")
}

func TestConsumeCubeFuel_amountが0以下なら消費せずtrue(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	a := addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(1*consts.MilligramPerKg))

	assert.True(t, lifecycle.ConsumeCubeFuel(world, cube, consts.Heat(0)), "0は消費不要で成功")
	assert.True(t, world.ECS.Alive(a), "0なら何も消費しない")
	assert.Equal(t, consts.Heat(800), query.CubeFuelTotal(world, cube))
}

func TestConsumeCubeFuel_不燃物は飛ばして燃料だけ消費する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := world.ECS.NewEntity()
	// 石は不燃で燃焼熱量0。収納に混ざっていても消費対象から外し、燃料だけ減らす
	stone := addCubeFuel(t, world, cube, oapi.STONE, consts.Milligram(5*consts.MilligramPerKg))
	coal := addCubeFuel(t, world, cube, oapi.COAL, consts.Milligram(1*consts.MilligramPerKg))

	assert.True(t, lifecycle.ConsumeCubeFuel(world, cube, consts.Heat(800)))
	assert.True(t, world.ECS.Alive(stone), "不燃物は消費されず残る")
	assert.False(t, world.ECS.Alive(coal), "燃料は消費される")
}
