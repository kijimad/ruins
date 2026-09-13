package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestDriveRangeTiles_手計算と一致(t *testing.T) {
	t.Parallel()
	// 燃料1000、積載0kg。燃費は基準10なので 1000/10 = 100 タイル。
	assert.InDelta(t, 100, DriveRangeTiles(1000, 0), 1e-9)
	// 積載90kg なら燃費 10+90=100。1000/100 = 10 タイル。積荷が航続を直接削る
	assert.InDelta(t, 10, DriveRangeTiles(1000, 90*consts.MilligramPerKg), 1e-9)
}

func TestDriveRangeAllFuel_自己ブレーキ(t *testing.T) {
	t.Parallel()
	// OIL 500kg: 熱量 1000*500=500000、燃費 10+500=510。500000/510 ≈ 980。
	assert.InDelta(t, 980.39, DriveRangeAllFuel(oapi.OIL, 500), 0.1)
	// WOOD 500kg: 200*500=100000 / 510 ≈ 196。材質の kg あたり熱量が航続上限をほぼ決める
	assert.InDelta(t, 196.08, DriveRangeAllFuel(oapi.WOOD, 500), 0.1)
	// 不燃材質は熱量0で航続0
	assert.Equal(t, 0.0, DriveRangeAllFuel(oapi.METAL, 500))
}
