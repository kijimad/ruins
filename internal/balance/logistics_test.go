package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestTreeFuelBalance(t *testing.T) {
	t.Parallel()
	master := loadTestMaster(t)
	player, err := LoadCombatantFromMember(master, "ash")
	require.NoError(t, err)
	bareHands, err := LoadWeaponFromItem(master, "bare_hands")
	require.NoError(t, err)

	t.Run("素手は確定枠だけ、斧は確率枠も得て作業あたりの燃料が3倍以上になる", func(t *testing.T) {
		t.Parallel()
		f, err := TreeFuelBalance(master, player, bareHands, "tree_a", 2)
		require.NoError(t, err)
		// 確定枠 1d2 の期待値1.5本、薪1本は100ターン燃える
		assert.InDelta(t, 150, f.HandBurnTurns, 1e-9)
		// 確率枠は 60+グレード補正10 で70%
		assert.InDelta(t, 150+0.7*150, f.ToolBurnTurns, 1e-9)
		// baseAP 1500 にグレード2の0.8倍、100APで1ターン
		assert.InDelta(t, 12, f.ToolTurns, 1e-9)
		assert.Greater(t, f.ToolBurnTurns/f.ToolTurns, 3*f.HandBurnTurns/f.HandTurns)
	})

	t.Run("分解定義のない prop はエラーを返す", func(t *testing.T) {
		t.Parallel()
		_, err := TreeFuelBalance(master, player, bareHands, "bathtub", 2)
		require.ErrorIs(t, err, ErrNoDisassembly)
	})

	t.Run("存在しない prop はエラーを返す", func(t *testing.T) {
		t.Parallel()
		_, err := TreeFuelBalance(master, player, bareHands, "no_such_prop", 2)
		var notFound raw.KeyNotFoundError
		require.ErrorAs(t, err, &notFound)
	})
}
