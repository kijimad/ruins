package lifecycle

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// AddCubeFuel は fuel の燃焼熱量をキューブの燃料ポイントへ足し、fuel を消す。焚き火の AddFuel と対になる。
func AddCubeFuel(world w.World, cube ecs.Entity, fuel ecs.Entity) {
	if !world.Components.FuelTank.Has(cube) {
		return
	}
	world.Components.FuelTank.Get(cube).Heat += query.HeatContent(world, fuel)
	world.ECS.RemoveEntity(fuel)
}

// ConsumeCubeFuel はキューブの燃料ポイントを amount 減らす。足りていれば減らして true、
// 足りなければ何も減らさず false を返す。
func ConsumeCubeFuel(world w.World, cube ecs.Entity, amount consts.Heat) bool {
	if amount <= 0 {
		return true
	}
	if query.CubeFuelTotal(world, cube) < amount {
		return false
	}
	world.Components.FuelTank.Get(cube).Heat -= amount
	return true
}
