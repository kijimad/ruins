package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestDriveRangeTiles_手計算と一致(t *testing.T) {
	t.Parallel()
	// 燃料1000、積載0kg。燃費は基準5なので 1000/5 = 200 タイル。
	assert.InDelta(t, 200, DriveRangeTiles(1000, 0), 1e-9)
	// 積載95kg なら燃費 5+95=100。1000/100 = 10 タイル。積荷が航続を直接削る
	assert.InDelta(t, 10, DriveRangeTiles(1000, 95*consts.MilligramPerKg), 1e-9)
}

func TestDriveRangeAllFuel_自己ブレーキ(t *testing.T) {
	t.Parallel()
	// OIL 500kg: 熱量 1000*500=500000、燃費 5+500=505。500000/505 ≈ 990。
	assert.InDelta(t, 990.10, DriveRangeAllFuel(oapi.OIL, 500), 0.1)
	// WOOD 500kg: 200*500=100000 / 505 ≈ 198。材質の kg あたり熱量が航続上限をほぼ決める
	assert.InDelta(t, 198.02, DriveRangeAllFuel(oapi.WOOD, 500), 0.1)
	// 不燃材質は熱量0で航続0
	assert.Equal(t, 0.0, DriveRangeAllFuel(oapi.METAL, 500))
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
