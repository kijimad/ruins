package components

import "github.com/kijimaD/ruins/internal/consts"

// HeatSource は周囲を暖める熱源の profile。効きは2系統ある。
// 近接するキャラの低体温タイマーを毎ターン直接回復し、あわせて周囲気温も押し上げる。
// どちらも距離に応じて線形に減衰し、源泉で Warmth 満額、半径の縁で最小になる。
// 暖房かどうかは HeatSource の有無だけで決まり、電熱のように燃えない熱源も暖房になる。
// 火は燃え尽きるとエンティティごと除去されるので数から外れる。
type HeatSource struct {
	Radius consts.Tile // 熱の到達半径。チェビシェフ距離
	Warmth float64     // 源泉で毎ターン下げる低体温タイマーの量
}
