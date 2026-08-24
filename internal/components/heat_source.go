package components

import "github.com/kijimaD/ruins/internal/consts"

// HeatSource は近接するキャラの低体温を毎ターン回復する熱源。
// 周囲気温は変えず、体温タイマーへ直接効く。回復量は屋内外で同じ。
type HeatSource struct {
	Radius consts.Tile // 熱の到達半径。チェビシェフ距離
	Warmth float64     // 毎ターン下げる低体温タイマーの量
}
