package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHeroCenterpiece_同じseedで完全一致する は hero の抽選の決定性を固定する。再訪一致と serde の前提。
func TestHeroCenterpiece_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	for seed := range uint64(20) {
		ref, ok := heroCenterpiece(seed)
		for range 5 {
			ref2, ok2 := heroCenterpiece(seed)
			require.Equalf(t, ok, ok2, "seed=%d の hero 有無が一致する", seed)
			require.Equalf(t, ref, ref2, "seed=%d の hero 種別が一致する", seed)
		}
	}
}

// TestHeroCenterpiece_稀に出て報酬が最多 は landmark が孤立して稀に出ること、種別が予算配分(報酬46/雰囲気25/
// リスクリワード15/危険14)に沿って報酬を最多にすることを多 seed で固定する。全建物に見せ場が出て陳腐化する
// 退行と、報酬が消える退行を止める。
func TestHeroCenterpiece_稀に出て報酬が最多(t *testing.T) {
	t.Parallel()

	const n = 3000
	kinds := map[string]int{}
	heroes := 0
	for seed := range uint64(n) {
		if ref, ok := heroCenterpiece(seed); ok {
			heroes++
			kinds[ref]++
		}
	}
	// 稀。全体の 1〜12% の建物だけが見せ場を持つ
	assert.Greaterf(t, heroes, n/100, "見せ場を持つ建物が稀に出る (%d/%d)", heroes, n)
	assert.Lessf(t, heroes, n*12/100, "見せ場は稀に留まる (%d/%d)", heroes, n)
	// 報酬(chest)が最多。予算配分の 46% を反映する
	for kind, c := range kinds {
		if kind == "chest" {
			continue
		}
		assert.GreaterOrEqualf(t, kinds["chest"], c, "報酬(chest %d)は %s(%d)以上出る", kinds["chest"], kind, c)
	}
}

// TestFurnishBuilding_hero建物は主室に目玉を置く は hero の建物が主室の中央へ landmark を1つ据えることを
// 固定する。seed 0 は hero(報酬の宝箱)。生成した配置に宝箱が含まれ、主室の内側にあることを守る。
func TestFurnishBuilding_hero建物は主室に目玉を置く(t *testing.T) {
	t.Parallel()

	ref, ok := heroCenterpiece(0)
	require.True(t, ok, "seed 0 は hero")

	site, placed := FurnishBuilding(0, Rect{X: 0, Y: 0, W: 20, H: 20}, Vec{X: 10, Y: 0}, "house")
	found := false
	for _, p := range placed {
		if p.Ref == ref {
			found = true
			break
		}
	}
	assert.Truef(t, found, "hero 建物の配置に目玉 %q が含まれる", ref)

	// 目玉は最大の部屋(廊下を除く)の内側に置かれる
	spot, ok := heroSpot(site)
	require.True(t, ok, "目玉の据え場所がある")
	inside := false
	for _, hr := range site.Rooms {
		if hr.Room.Rect.containsInterior(spot) {
			inside = true
			break
		}
	}
	assert.True(t, inside, "目玉は部屋の内側に置かれる")
}
