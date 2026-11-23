package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// TestWinnerOf_SpacingがSeparation以下はパニックする は、不正な Placement 設定を never として
// 弾くことを固定する。差が負になり uint64 アンダーフローで巨大値を引く退行を検知する。
func TestWinnerOf_SpacingがSeparation以下はパニックする(t *testing.T) {
	t.Parallel()

	p := Placement{Spacing: 2, Separation: 2, Salt: 1}
	assert.PanicsWithValue(t, "Placement: Spacing must be greater than Separation", func() {
		p.WinnerOf(1, 0, 9)
	}, "Spacing <= Separation の設定ミスはパニックで弾く")
}

func TestPlacements_実データの全ての地物はSpacingがSeparationより大きい(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	for _, kind := range rawschema.PointFeatureKindValues() {
		p := placementOf(raws, kind)
		assert.Greaterf(t, p.Spacing, p.Separation, "%s の Spacing は Separation より大きい", kind)
	}
}

func TestPlacements_地物の行のソルトは区域と散布のソルトと重ならない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	got := map[uint64]bool{districtSalt: true, scatterSalt: true, outdoorLootSalt: true, crashSiteSalt: true, crossingSalt: true, settlementSiteSalt: true}
	for _, kind := range rawschema.PointFeatureKindValues() {
		salt := placementOf(raws, kind).Salt
		assert.Falsef(t, got[salt], "%s のソルト %d が他と重なる", kind, salt)
		got[salt] = true
	}
}
