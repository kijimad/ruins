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

// DaysUntilStarving は満腹から栄養失調に入るまでの日数を返す。空腹度が飢餓しきい値を割ると栄養失調になる。
// しきい値は components の単一出典 HungerStarvingRatio を参照し、ゲーム側の変更に追従する。
func DaysUntilStarving() float64 {
	lost := float64(gc.DefaultMaxHunger) * (1 - gc.HungerStarvingRatio)
	turns := lost / hungerDrainPerTurn()
	return turns / float64(gc.TurnsPerDay)
}

// HPDrainPerTurnAtBlood は血液量から失血による1ターンのHP減少量を返す。状態異常→血液低下→HP減の
// 連鎖の終端。血液は components.BloodLossHPDrainRate の単一出典を参照する。
func HPDrainPerTurnAtBlood(blood int) int {
	return gc.BloodLossHPDrainRate(blood)
}

// DaysUntilTired は起床し続けて疲労状態になるまでの日数を返す。疲労は毎ターン一定量たまる。
func DaysUntilTired() float64 {
	turns := gc.FatigueTiredRatio * float64(gc.DefaultMaxFatigue) / float64(gc.FatigueGainPerTurn)
	return turns / float64(gc.TurnsPerDay)
}

// DaysUntilExhausted は起床し続けて過労状態になるまでの日数を返す。
func DaysUntilExhausted() float64 {
	turns := gc.FatigueExhaustedRatio * float64(gc.DefaultMaxFatigue) / float64(gc.FatigueGainPerTurn)
	return turns / float64(gc.TurnsPerDay)
}

// TurnsToHypothermia は実効温度で低体温が発生し始めるまでのターン数を返す。実効温度は
// 周囲温度に断熱を足した値。体温は systems.CalcBodyTempRate の速さで平熱から冷え、低体温帯
// BodyTempColdBand を割るとタイマーが進み始める。
// rate は温まる向きが正・冷える向きが負。冷えない適温すなわち rate >= 0 では低体温に至らず 0 を返す。
func TurnsToHypothermia(effectiveTemp int) float64 {
	rate := systems.CalcBodyTempRate(effectiveTemp)
	if rate >= 0 {
		return 0
	}
	return systems.BodyTempColdBand / -rate
}
