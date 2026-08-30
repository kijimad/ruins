package components

import "github.com/kijimaD/ruins/internal/consts"

// HeatSource は近接するキャラの低体温を毎ターン回復する熱源の profile。
// 周囲気温は変えず、体温タイマーへ直接効く。回復量は屋内外で同じ。
// 効きは距離に応じて線形に減衰し、源泉で Warmth 満額、半径の縁で最小になる。
// 暖房として効くのは Burning を併せ持つ、すなわち今燃えているエンティティのときだけ。
// 火が消えたタイルは HeatSource を残しても heatSourceWarmthAt に数えられない。
type HeatSource struct {
	Radius consts.Tile // 熱の到達半径。チェビシェフ距離
	Warmth float64     // 源泉で毎ターン下げる低体温タイマーの量
}
