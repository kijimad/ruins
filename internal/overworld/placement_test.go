package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

// TestFloorDiv は負の座標でもリージョン割りが床方向に連続することを固定する。
// Go の / はゼロ方向へ丸めるため、負側で境界が二重にならないことを検証する。
func TestFloorDiv(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b consts.Chunk
		want consts.Chunk
	}{
		{"正で割り切れる", 6, 3, 2},
		{"正で余りあり", 7, 3, 2},
		{"ゼロ", 0, 3, 0},
		{"負で割り切れる", -3, 3, -1},
		{"負で余りあり床方向へ", -1, 3, -1},
		{"負で余りあり床方向へ2", -4, 3, -2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.want, floorDiv(c.a, c.b), "床除算がリージョン境界で連続する")
		})
	}
}

// TestWinnerOf_SpacingがSeparation以下はパニックする は、不正な Placement 設定を never として
// 弾くことを固定する。差が負になり uint64 アンダーフローで巨大値を引く退行を検知する。
func TestWinnerOf_SpacingがSeparation以下はパニックする(t *testing.T) {
	t.Parallel()

	p := Placement{Spacing: 2, Separation: 2, Salt: 1}
	assert.PanicsWithValue(t, "Placement: Spacing は Separation より大きいこと", func() {
		p.WinnerOf(1, 0, 9)
	}, "Spacing <= Separation の設定ミスはパニックで弾く")
}

// TestPlacements_全登録配置はSpacingがSeparationより大きい は、登録済みの Placement が
// WinnerOf のガードに引っかからない健全な設定であることを固定する。
func TestPlacements_全登録配置はSpacingがSeparationより大きい(t *testing.T) {
	t.Parallel()

	for name, p := range map[string]Placement{
		"settlement": settlementPlacement,
		"urban":      urbanPlacement,
		"ruin":       ruinPlacement,
		"poi":        poiPlacement,
	} {
		assert.Greaterf(t, p.Spacing, p.Separation, "%s の Spacing は Separation より大きい", name)
	}
}
