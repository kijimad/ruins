package balance

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/systems"
)

// hungerDrainPerTurn は空腹進行が基準100%のときの1ターンあたり満腹度減耗を返す。
// systems/hunger.go の確率ゲート pct/(PercentBase*HungerDrainTurns) は、pct=PercentBase の基準で
// 1/HungerDrainTurns に約分される。その期待値を導出に使う。
func hungerDrainPerTurn() float64 {
	return 1.0 / float64(gc.HungerDrainTurns)
}

// DaysUntilHungerEmpty は満腹から補給なしで満腹度が尽きるまでの日数を返す。
func DaysUntilHungerEmpty() float64 {
	turns := float64(gc.DefaultMaxHunger) / hungerDrainPerTurn()
	return turns / float64(gc.TurnsPerDay)
}

// hungerStarvingRatio は空腹度がこの割合を下回ると栄養失調に入るしきい値。
// components/hunger.go の GetLevel が HungerStarving を返す 33% に一致させる。
const hungerStarvingRatio = 0.33

// DaysUntilStarving は満腹から栄養失調に入るまでの日数を返す。空腹度が33%を割ると栄養失調になる。
func DaysUntilStarving() float64 {
	lost := float64(gc.DefaultMaxHunger) * (1 - hungerStarvingRatio)
	turns := lost / hungerDrainPerTurn()
	return turns / float64(gc.TurnsPerDay)
}

// TurnsToHypothermia は実効温度で低体温が発生し始めるまでのターン数を返す。実効温度は
// 周囲温度に断熱を足した値。体温は systems.CalcBodyTempRate の速さで平熱から冷え、低体温帯
// BodyTempColdBand を割るとタイマーが進み始める。冷却が起きない適温では 0 を返す。
func TurnsToHypothermia(effectiveTemp int) float64 {
	rate := systems.CalcBodyTempRate(effectiveTemp)
	if rate >= 0 {
		return 0
	}
	return systems.BodyTempColdBand / -rate
}
