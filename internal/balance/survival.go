package balance

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/systems"
)

// hungerDrainPerTurn は空腹進行が基準100%のときの1ターンあたり満腹度減耗を返す。systems の確率ゲートは基準で
// 1/HungerDrainTurns に約分され、その期待値を導出に使う。
func hungerDrainPerTurn(p Params) float64 {
	return 1.0 / p.HungerDrainTurns
}

// DaysUntilHungerEmpty は満腹から補給なしで満腹度が尽きるまでの日数を返す。
func DaysUntilHungerEmpty(p Params) float64 {
	turns := p.MaxHunger / hungerDrainPerTurn(p)
	return turns / p.TurnsPerDay
}

// DaysUntilStarving は満腹から栄養失調に入るまでの日数を返す。減耗は1ターン 1/HungerDrainTurns なので、飢餓しきい値まで
// 失う量に HungerDrainTurns を掛けてターン数にする。
func DaysUntilStarving(p Params) float64 {
	lost := p.MaxHunger * (1 - p.HungerStarvingRatio)
	turns := lost * p.HungerDrainTurns
	return turns / p.TurnsPerDay
}

// HPDrainPerTurnAtBlood は血液量から失血による1ターンのHP減少量を返す。状態異常→血液低下→HP減の
// 連鎖の終端。血液は components.BloodLossHPDrainRate の単一出典を参照する。
func HPDrainPerTurnAtBlood(blood int) int {
	return gc.BloodLossHPDrainRate(blood)
}

// DaysUntilTired は起床し続けて疲労状態になるまでの日数を返す。疲労は毎ターン一定量たまる。
func DaysUntilTired(p Params) float64 {
	turns := p.FatigueTiredRatio * p.MaxFatigue / p.FatigueGainPerTurn
	return turns / p.TurnsPerDay
}

// DaysUntilExhausted は起床し続けて過労状態になるまでの日数を返す。
func DaysUntilExhausted(p Params) float64 {
	turns := p.FatigueExhaustedRatio * p.MaxFatigue / p.FatigueGainPerTurn
	return turns / p.TurnsPerDay
}

// SleepTurnsToFullRecover は満タンの疲労を地べた基準の睡眠で回復し切るターン数を返す。整数回復で0以下になった時点で
// 目覚めるため、実ターン数はこの値の切り上げ。
func SleepTurnsToFullRecover(p Params) float64 {
	return p.MaxFatigue / p.FatigueRecoverPerTurn
}

// SleepTimeFraction は疲労を釣り合わせるのに時間のどれだけを睡眠へ充てるかの割合を返す。起床は FatigueGainPerTurn 溜まり
// 睡眠は FatigueRecoverPerTurn 抜けるので、gain/(gain+recover) が定常の睡眠時間比になる。
func SleepTimeFraction(p Params) float64 {
	return p.FatigueGainPerTurn / (p.FatigueGainPerTurn + p.FatigueRecoverPerTurn)
}

// TurnsToHypothermia は実効温度で低体温が始まるまでのターン数を返す。実効温度は周囲温度に断熱を足した値。
// rate は温まる向きが正・冷える向きが負で、冷えない適温 rate>=0 では低体温に至らず 0 を返す。
func TurnsToHypothermia(effectiveTemp int) float64 {
	rate := systems.CalcBodyTempRate(effectiveTemp)
	if rate >= 0 {
		return 0
	}
	return systems.BodyTempColdBand / -rate
}
