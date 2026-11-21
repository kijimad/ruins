package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ChargeRatio は残量の上限に対する割合を 0 から 1 で返す。残量を持たなければ ok は偽。上限は raw で1以上に決まる
func ChargeRatio(world w.World, entity ecs.Entity) (float64, bool) {
	if !world.Components.Charge.Has(entity) {
		return 0, false
	}
	charge := world.Components.Charge.Get(entity)
	return max(0, min(1, float64(charge.Current)/float64(charge.Max))), true
}
