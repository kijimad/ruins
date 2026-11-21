package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// ChargeStageCount は残量を見せる段階の数
const ChargeStageCount = 5

// ChargeStage は残量を ChargeStageCount 段のうち何段残っているかで返す。上限を ChargeStageCount 等分した境目を割るたびに
// 1段下がり、0 は尽きている。残量を持たなければ ok は偽
func ChargeStage(world w.World, entity ecs.Entity) (int, bool) {
	if !world.Components.Charge.Has(entity) {
		return 0, false
	}
	charge := world.Components.Charge.Get(entity)
	return chargeStage(charge.Current, charge.Max), true
}

// chargeStage は current と max から段階を求める。割り切れないときは切り上げ、少しでも残れば1段になる
func chargeStage(current, maxCharge int) int {
	if current <= 0 || maxCharge <= 0 {
		return 0
	}
	return min((current*ChargeStageCount+maxCharge-1)/maxCharge, ChargeStageCount)
}
