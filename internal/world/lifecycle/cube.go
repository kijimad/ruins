package lifecycle

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ConsumeCubeFuel はキューブ収納の燃料を amount 分だけ消費する。足りていれば消費して true、
// 足りなければ何も消費せず false を返す。判定を先に済ませ、部分消費で中途半端に減らさない。
// 燃料は丸ごと単位で消費し、必要量に達したら止める。端数はその1個を使い切る。
func ConsumeCubeFuel(world w.World, cube ecs.Entity, amount consts.Heat) bool {
	if amount <= 0 {
		return true
	}
	if query.CubeFuelTotal(world, cube) < amount {
		return false
	}
	// GetStorageItems は収集済みスライスを返すので、反復中の RemoveEntity は安全
	var consumed consts.Heat
	for _, item := range query.GetStorageItems(world, cube) {
		if consumed >= amount {
			break
		}
		h := query.HeatContent(world, item)
		if h <= 0 {
			continue
		}
		consumed += h
		world.ECS.RemoveEntity(item)
	}
	return true
}
