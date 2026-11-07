package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestFollowRangeTiles_手計算と一致(t *testing.T) {
	t.Parallel()
	// 燃料1000、積載0kg。燃費は基準5なので 1000/5 = 200 歩。
	assert.InDelta(t, 200, FollowRangeTiles(1000, 0), 1e-9)
	// 積載95kg なら燃費 5+95=100。1000/100 = 10 歩。積荷が航続を直接削る
	assert.InDelta(t, 10, FollowRangeTiles(1000, 95*consts.MilligramPerKg), 1e-9)
}

func TestFollowRangeFullTank_満タンの燃料ポイントで決まる(t *testing.T) {
	t.Parallel()
	// 満タン3000、積荷0kgの燃費5で 600 歩
	assert.InDelta(t, 600, FollowRangeFullTank(0), 1e-9)
	// 積荷95kgなら燃費100で 30 歩
	assert.InDelta(t, 30, FollowRangeFullTank(95*consts.MilligramPerKg), 1e-9)
}

func TestFuelBurnTurns_熱量と効率で決まる(t *testing.T) {
	t.Parallel()
	// WOOD 1kg: 熱量200、地面効率50%で 200*50/100 = 100 ターン。重量に比例する。
	assert.Equal(t, 100, FuelBurnTurns(oapi.WOOD, 1))
	assert.Equal(t, 1000, FuelBurnTurns(oapi.WOOD, 10))
	// OIL は kg あたり熱量が高いので同じ重さでも長く燃える
	assert.Greater(t, FuelBurnTurns(oapi.OIL, 10), FuelBurnTurns(oapi.WOOD, 10))
	// 不燃材質は0
	assert.Equal(t, 0, FuelBurnTurns(oapi.METAL, 10))
}
