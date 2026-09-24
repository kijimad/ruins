package components

import "github.com/kijimaD/ruins/internal/consts"

// HeatSource は周囲を暖める熱源の profile。効きは2系統ある。
// 近接するキャラの低体温タイマーを毎ターン直接回復し、あわせて周囲気温も押し上げる。
// どちらも半径内では距離によらず一律に効き、半径外は効かない。
// 暖房かどうかは HeatSource の有無だけで決まり、電熱のように燃えない熱源も暖房になる。
// 火は燃え尽きるとエンティティごと除去されるので数から外れる。
type HeatSource struct {
	Radius consts.Tile // 一律に暖まる到達半径。チェビシェフ距離
	Warmth float64     // 半径内で毎ターン下げる低体温タイマーの量
}
