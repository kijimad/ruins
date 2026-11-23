package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFollowRangeTiles_手計算と一致(t *testing.T) {
	t.Parallel()
	// 燃料1000、積載0kg。燃費は基準5なので 1000/5 = 200 歩。
	assert.InDelta(t, 200, FollowRangeTiles(1000, 0), 1e-9)
	// 積載95kg なら燃費 5+95=100。1000/100 = 10 歩。積荷が航続を直接削る
	assert.InDelta(t, 10, FollowRangeTiles(1000, 95*consts.MilligramPerKg), 1e-9)
}

func TestFollowRangeTiles_燃費が0以下なら0除算を避けて0(t *testing.T) {
	t.Parallel()
	// 燃費が0になる負の積載。基準燃料を1kgあたりの燃料で打ち消す
	require.Zero(t, consts.FollowFuelBase%consts.FollowFuelPerKg, "燃費がちょうど0になる整数kgが存在する前提")
	zeroKg := consts.Milligram(-consts.FollowFuelBase / consts.FollowFuelPerKg)
	assert.Equal(t, 0.0, FollowRangeTiles(1000, zeroKg*consts.MilligramPerKg), "燃費がちょうど0")
	assert.Equal(t, 0.0, FollowRangeTiles(1000, (zeroKg-1)*consts.MilligramPerKg), "燃費が負")
}

func TestFuelBurnTurns_熱量と効率で決まる(t *testing.T) {
	t.Parallel()
	defs, err := raw.NewMaterialDefs(loadTestMaster(t))
	require.NoError(t, err)
	// wood 1kg: 熱量200、地面効率50%で 200*50/100 = 100 ターン。重量に比例する。
	assert.Equal(t, 100, FuelBurnTurns(defs, "wood", 1))
	assert.Equal(t, 1000, FuelBurnTurns(defs, "wood", 10))
	// OIL は kg あたり熱量が高いので同じ重さでも長く燃える
	assert.Greater(t, FuelBurnTurns(defs, "oil", 10), FuelBurnTurns(defs, "wood", 10))
	// 不燃材質は0
	assert.Equal(t, 0, FuelBurnTurns(defs, "metal", 10))
}
