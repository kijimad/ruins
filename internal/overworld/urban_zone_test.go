package overworld

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUrbanSizeOf_各辺は2からurbanMaxSpanの範囲 は、市街地の縦横チャンク数が常に 2..urbanMaxSpan に
// 収まることを固定する。範囲外だと zoneOf の中心判定や規模 gate の前提が崩れる。
func TestUrbanSizeOf_各辺は2からurbanMaxSpanの範囲(t *testing.T) {
	t.Parallel()

	for s := range uint64(1000) {
		w, h := urbanSizeOf(s)
		assert.GreaterOrEqualf(t, w, consts.Chunk(2), "seed=%d の幅", s)
		assert.LessOrEqualf(t, w, urbanMaxSpan, "seed=%d の幅", s)
		assert.GreaterOrEqualf(t, h, consts.Chunk(2), "seed=%d の高さ", s)
		assert.LessOrEqualf(t, h, urbanMaxSpan, "seed=%d の高さ", s)
	}
}

// TestZoneOf_都心は奇数辺市街地の中心にだけ出る は、都心が 3×3 の厳密な中心にだけ現れ、偶数辺の
// 市街地には現れないことを固定する。都心にだけ専門施設を集める設計の前提。
func TestZoneOf_都心は奇数辺市街地の中心にだけ出る(t *testing.T) {
	t.Parallel()

	assert.Equal(t, zoneDowntown, zoneOf(1, 1, 3, 3, 0), "3×3 の中心 (1,1) は都心")

	for _, c := range []consts.Coord[consts.Chunk]{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 1}, {X: 2, Y: 2}} {
		assert.NotEqualf(t, zoneDowntown, zoneOf(c.X, c.Y, 3, 3, 0), "3×3 の非中心 %v は都心でない", c)
	}

	// 偶数辺 2×2 には中心マスが存在せず、どのマスも都心にならない
	for lx := range consts.Chunk(2) {
		for ly := range consts.Chunk(2) {
			assert.NotEqualf(t, zoneDowntown, zoneOf(lx, ly, 2, 2, 0), "2×2 の (%d,%d) は都心でない", lx, ly)
		}
	}
}

// TestZoneOf_都心以外はビットで産業区と住宅地に分かれる は、都心でないマスが市街地の性格ビットで
// 産業区と住宅地に排他的に振り分けられることを固定する。
func TestZoneOf_都心以外はビットで産業区と住宅地に分かれる(t *testing.T) {
	t.Parallel()

	assert.Equal(t, zoneIndustrial, zoneOf(0, 0, 3, 3, industrialUrbanBit), "ビット立ちは産業区")
	assert.Equal(t, zoneResidential, zoneOf(0, 0, 3, 3, 0), "ビット無しは住宅地")
}

// TestRollFacilityInZone_規模gateで専門施設はspan3以上でだけ出る は、規模 gate が効いて小さな市街地に
// 専門施設が混ざらないことを固定する。都心の骨董品店・診療所・研究施設は minSpan=3 なので span=2 では
// 決して出ず、span=3 で初めて混ざる。gate を外すと小規模に専門施設が漏れる退行を検知する。
func TestRollFacilityInZone_規模gateで専門施設はspan3以上でだけ出る(t *testing.T) {
	t.Parallel()

	small := map[facilityType]bool{}
	big := map[facilityType]bool{}
	for s := range uint64(300) {
		small[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), zoneDowntown, 2)] = true
		big[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), zoneDowntown, 3)] = true
	}
	for _, f := range []facilityType{facilityAntique, facilityClinic, facilityLab} {
		assert.Falsef(t, small[f], "span=2 の都心に minSpan=3 の %s は出ない", f)
	}
	assert.True(t, big[facilityAntique] || big[facilityClinic] || big[facilityLab],
		"span=3 の都心では専門施設が混ざる")
}

// TestRollFacilityInZone_決定的 は、同じ rng 状態から同じ施設が出て、必ず有効な種別を返すことを固定する。
func TestRollFacilityInZone_決定的(t *testing.T) {
	t.Parallel()

	first := rollFacilityInZone(rand.New(rand.NewPCG(7, 0)), zoneResidential, 3)
	assert.NotEmpty(t, first, "有効な施設種別を返す")
	for range 5 {
		got := rollFacilityInZone(rand.New(rand.NewPCG(7, 0)), zoneResidential, 3)
		assert.Equal(t, first, got, "同じ rng シードなら同じ施設")
	}
}

// TestUrbanZoning_隣接同種率が独立期待を上回る は、ゾーニングで隣接チャンクの施設が空間相関を
// 持つことを固定する。per-chunk 独立抽選なら隣接同種率は施設頻度の二乗和すなわち独立期待に
// 一致するが、地区で重みを揃えるとそれを有意に上回る。相関ゼロのごま塩への退行を検知する。
func TestUrbanZoning_隣接同種率が独立期待を上回る(t *testing.T) {
	t.Parallel()

	const rows consts.Chunk = 9
	kindCount := map[facilityType]int{}
	total := 0
	sameAdj, totalAdj := 0, 0

	for s := uint64(1); s <= 120; s++ {
		grid := map[consts.Coord[consts.Chunk]]facilityType{}
		for y := range rows {
			for x := range consts.Chunk(60) {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if k, _, ok := urbanChunkInfo(s, c, rows); ok {
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
