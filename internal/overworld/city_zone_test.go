package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCityZoning_隣接同種率が独立期待を上回る は、ゾーニングで隣接チャンクの施設が空間相関を
// 持つことを固定する。per-chunk 独立抽選なら隣接同種率は施設頻度の二乗和すなわち独立期待に
// 一致するが、地区で重みを揃えるとそれを有意に上回る。相関ゼロのごま塩への退行を検知する。
func TestCityZoning_隣接同種率が独立期待を上回る(t *testing.T) {
	t.Parallel()

	const rows consts.Chunk = 9
	kindCount := map[facilityKind]int{}
	total := 0
	sameAdj, totalAdj := 0, 0

	for s := uint64(1); s <= 120; s++ {
		grid := map[consts.Coord[consts.Chunk]]facilityKind{}
		for y := range rows {
			for x := range consts.Chunk(60) {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if k, _, ok := cityChunkInfo(s, c, rows); ok {
					grid[c] = k
					kindCount[k]++
					total++
				}
			}
		}
		// 直交隣接の同種を数える。東と南だけ見て各ペアを一度ずつ数える
		for c, k := range grid {
			for _, n := range []consts.Coord[consts.Chunk]{{X: c.X + 1, Y: c.Y}, {X: c.X, Y: c.Y + 1}} {
				if nk, ok := grid[n]; ok {
					totalAdj++
					if nk == k {
						sameAdj++
					}
				}
			}
		}
	}

	require.Positive(t, totalAdj, "前提: 隣接する市街地チャンクのペアがある")
	observed := float64(sameAdj) / float64(totalAdj)
	// 独立期待は施設頻度の二乗和。近傍が独立に抽選されるなら observed はこれに一致する
	expected := 0.0
	for _, n := range kindCount {
		p := float64(n) / float64(total)
		expected += p * p
	}
	t.Logf("隣接同種率 observed=%.3f 独立期待=%.3f 市街地チャンク=%d ペア=%d", observed, expected, total, totalAdj)
	assert.Greater(t, observed, expected+0.05, "ゾーニングで隣接同種率が独立期待を有意に上回る")
}
