package lifecycle

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// AddCubeFuel は fuel の燃焼熱量を上限までキューブの燃料ポイントへ足して fuel を消す。満タンなら何もせず false を返す。
func AddCubeFuel(world w.World, cube ecs.Entity, fuel ecs.Entity) bool {
	if !world.Components.FuelTank.Has(cube) {
		return false
	}
	tank := world.Components.FuelTank.Get(cube)
	if tank.Current >= tank.Max {
		return false
	}
	tank.Current = min(tank.Max, tank.Current+query.HeatContent(world, fuel))
	world.ECS.RemoveEntity(fuel)
	return true
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
	world.Components.FuelTank.Get(cube).Current -= amount
	return true
}
