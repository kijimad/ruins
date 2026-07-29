package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRollProfile_同じseedで完全一致する は生活感プロファイルの決定性を固定する。再訪一致と serde の前提。
func TestRollProfile_同じseedで完全一致する(t *testing.T) {
	t.Parallel()

	for seed := range uint64(20) {
		first := rollProfile(seed)
		for range 5 {
			require.Equalf(t, first, rollProfile(seed), "seed=%d は同じプロファイルを引く", seed)
		}
	}
}

// TestRollProfile_直交軸がseedで散らばる は変種・散らかり・損傷の各軸が seed を振ると分布することを固定する。
// 標準が大多数・稀に事情のある建物、損傷は3分の2、散らかりは1割が汚部屋という
// 分布のレンジ内であることを多 seed で確かめる。全建物が一律になる退行を止める。
func TestRollProfile_直交軸がseedで散らばる(t *testing.T) {
	t.Parallel()

	const n = 2000
	var damageSeen [3]int
	var clutterSeen [3]int
	variantSeen := map[variantKind]int{}
	for seed := range uint64(n) {
		p := rollProfile(seed)
		damageSeen[p.damage]++
		clutterSeen[p.clutter]++
		variantSeen[p.variant]++
	}

	// 損傷は3段すべて出て、無傷は少数派、大破も出る。無傷2:小破3:大破1 の翻案
	for lvl := range damageSeen {
		assert.Positivef(t, damageSeen[lvl], "損傷レベル %d が出る", lvl)
	}
	assert.Less(t, damageSeen[dmgIntact], damageSeen[dmgMinor], "無傷は小破より少ない")

	// 散らかりは3段すべて出て、整頓が最多、汚部屋は少数
	for lvl := range clutterSeen {
		assert.Positivef(t, clutterSeen[lvl], "散らかりレベル %d が出る", lvl)
	}
	assert.Greater(t, clutterSeen[clutterTidy], clutterSeen[clutterFilthy], "整頓が汚部屋より多い")

	// 変種は標準が大多数。稀に事情のある建物が出る。2000 seed なら非標準が少なくとも1つは出る
	assert.Greater(t, variantSeen[varStandard], n*8/10, "変種は標準が大多数")
	assert.Less(t, variantSeen[varStandard], n, "稀に非標準の建物が出る")
}

// TestApplyClutter_整頓は何も足さず戸口を塞がない は散らかりの下限と安全を固定する。整頓では小物を足さず、
// 散らかった部屋でも戸口前には小物を置かないので、通行が塞がれない。
func TestApplyClutter_整頓は何も足さず戸口を塞がない(t *testing.T) {
	t.Parallel()

	room := storeRoom()
	base := FillRoom(1, room, storeContent())

	assert.Len(t, applyClutter(1, room, base, clutterTidy, "main"), len(base), "整頓では小物を足さない")

	filthy := applyClutter(1, room, base, clutterFilthy, "main")
	assert.Greater(t, len(filthy), len(base), "汚部屋では小物が増える")
	for _, p := range filthy {
		assert.Falsef(t, isDoorwayAdjacent(room, p.Pos), "小物 %v は戸口前に置かない", p.Pos)
	}
}
