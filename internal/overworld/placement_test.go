package overworld

import (
	"testing"

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

// TestPlacements_全登録配置はSpacingがSeparationより大きい は、登録済みの Placement が
// WinnerOf のガードに引っかからない健全な設定であることを固定する。
func TestPlacements_全登録配置はSpacingがSeparationより大きい(t *testing.T) {
	t.Parallel()

	for name, p := range map[string]Placement{
		"settlement":       settlementPlacement,
		"urban":            urbanPlacement,
		"dungeon_entrance": dungeonEntrancePlacement,
		"landmark":         landmarkPlacement,
	} {
		assert.Greaterf(t, p.Spacing, p.Separation, "%s の Spacing は Separation より大きい", name)
	}
}
