package components

import (
	"github.com/kijimaD/ruins/internal/consts"
)

// Fuel は燃やせるものが持つ性質。燃やしたときの熱量を表す。木・紙・油で値が違う。
// 可燃性はこのコンポーネントの有無で判定し、燃えないものは持たない。
type Fuel struct {
	HeatContent consts.Heat // 燃やしたとき効率で割り引いて Burning の残量へ足す熱量。常に正
}
