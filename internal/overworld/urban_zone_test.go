package overworld

import (
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZoneOf_大きい街ほど中心の都心が広い(t *testing.T) {
	t.Parallel()

	downtowns := func(w, h consts.Chunk) []consts.Coord[consts.Chunk] {
		var out []consts.Coord[consts.Chunk]
		for ly := range h {
			for lx := range w {
				if zoneOf(lx, ly, w, h, 0) == rawschema.Downtown {
					out = append(out, consts.Coord[consts.Chunk]{X: lx, Y: ly})
				}
			}
		}
		return out
	}
	assert.Empty(t, downtowns(2, 2))
	assert.Equal(t, []consts.Coord[consts.Chunk]{{X: 1, Y: 1}}, downtowns(3, 3))
	assert.Empty(t, downtowns(4, 3), "中心の列が無い")
	assert.Equal(t, []consts.Coord[consts.Chunk]{{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 1, Y: 2}, {X: 2, Y: 2}}, downtowns(4, 4))
	assert.Equal(t, []consts.Coord[consts.Chunk]{
		{X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3, Y: 1},
		{X: 1, Y: 2}, {X: 2, Y: 2}, {X: 3, Y: 2},
		{X: 1, Y: 3}, {X: 2, Y: 3}, {X: 3, Y: 3},
	}, downtowns(5, 5))
}

// TestZoneOf_都心以外はビットで産業区と住宅地に分かれる は、都心でないマスが市街地の性格ビットで
// 産業区と住宅地に排他的に振り分けられることを固定する。
func TestZoneOf_都心以外はビットで産業区と住宅地に分かれる(t *testing.T) {
	t.Parallel()

	assert.Equal(t, rawschema.Industrial, zoneOf(0, 0, 3, 3, industrialUrbanBit), "ビット立ちは産業区")
	assert.Equal(t, rawschema.Residential, zoneOf(0, 0, 3, 3, 0), "ビット無しは住宅地")
}

// TestRollFacilityInZone_規模gateで専門施設はspan3以上でだけ出る は、規模 gate が効いて小さな市街地に
// 専門施設が混ざらないことを固定する。都心の骨董品店・診療所は minSpan=3 なので span=2 では
// 決して出ず、span=3 で初めて混ざる。gate を外すと小規模に専門施設が漏れる退行を検知する。
func TestRollFacilityInZone_規模gateで専門施設はspan3以上でだけ出る(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	small := map[string]bool{}
	big := map[string]bool{}
	for s := range uint64(300) {
		small[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), cat, rawschema.Downtown, 2)] = true
		big[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), cat, rawschema.Downtown, 3)] = true
	}
	for _, f := range []string{"antique", "clinic"} {
		assert.Falsef(t, small[f], "span=2 の都心に minSpan=3 の %s は出ない", f)
	}
	assert.True(t, big["antique"] || big["clinic"],
		"span=3 の都心では専門施設が混ざる")
}

// TestRollFacilityInZone_決定的 は、同じ rng 状態から同じ施設が出て、必ず有効な種別を返すことを固定する。
func TestRollFacilityInZone_決定的(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	first := rollFacilityInZone(rand.New(rand.NewPCG(7, 0)), cat, rawschema.Residential, 3)
	assert.NotEmpty(t, first, "有効な施設種別を返す")
	for range 5 {
		got := rollFacilityInZone(rand.New(rand.NewPCG(7, 0)), cat, rawschema.Residential, 3)
		assert.Equal(t, first, got, "同じ rng シードなら同じ施設")
	}
}

// TestUrbanZoning_隣接同種率が独立期待を上回る は、ゾーニングで隣接チャンクの施設が空間相関を
// 持つことを固定する。per-chunk 独立抽選なら隣接同種率は施設頻度の二乗和すなわち独立期待に
// 一致するが、地区で重みを揃えるとそれを有意に上回る。相関ゼロのごま塩への退行を検知する。
func TestUrbanZoning_隣接同種率が独立期待を上回る(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	const rows consts.Chunk = 9
	kindCount := map[string]int{}
	total := 0
	sameAdj, totalAdj := 0, 0

	for s := uint64(1); s <= 120; s++ {
		grid := map[consts.Coord[consts.Chunk]]string{}
		for y := range rows {
			for x := range consts.Chunk(60) {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if r, ok := districtAt(raws, s, c, rows); ok {
					k := urbanFacilityAt(cat, r, c)
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

func TestCampusZoneOf_南の門から北の奥へ地区が並ぶ(t *testing.T) {
	t.Parallel()

	got := make([]rawschema.Zone, 6)
	for ly := range consts.Chunk(6) {
		got[ly] = campusZoneOf(ly, 6)
	}
	assert.Equal(t, []rawschema.Zone{
		rawschema.CampusQuarantine,
		rawschema.CampusInner, rawschema.CampusInner, rawschema.CampusInner, rawschema.CampusInner,
		rawschema.CampusGate,
	}, got)
}

func TestDistrictRegion_zoneAt_区域の作りが割り振る地区はrawの地区の一覧に収まる(t *testing.T) {
	t.Parallel()

	for _, layout := range rawschema.DistrictLayoutValues() {
		got := map[rawschema.Zone]bool{}
		for _, size := range [][2]consts.Chunk{{2, 2}, {3, 3}, {4, 3}, {5, 5}, {7, 8}} {
			for seed := range uint64(4) {
				r := districtRegion{layout: layout, w: size[0], h: size[1], seed: seed << 32}
				for ly := range r.h {
					for lx := range r.w {
						got[r.zoneAt(consts.Coord[consts.Chunk]{X: lx, Y: ly})] = true
					}
				}
			}
		}
		assert.ElementsMatch(t, raw.DistrictZones(layout), slices.Collect(maps.Keys(got)), layout)
	}
}

func TestCampusFence_外周を巡り真ん中の列の南北に門を開ける(t *testing.T) {
	t.Parallel()

	const w, h consts.Tile = 6, 5
	r := districtRegion{layout: rawschema.DistrictLayoutCampus, w: 3, h: 2}
	render := func(l consts.Coord[consts.Chunk]) []string {
		rows := make([][]byte, h)
		for y := range rows {
			rows[y] = []byte("......")
		}
		for _, p := range campusFence(r, l, w, h) {
			rows[p.Y][p.X] = 'f'
		}
		out := make([]string, h)
		for y := range rows {
			out[y] = string(rows[y])
		}
		return out
	}
	// 街路の幅 urbanStreetW は4。真ん中の列 lx=1 の北の端の行と南の端の行だけ、西の4マスを開ける
	assert.Equal(t, []string{"ffffff", "f.....", "f.....", "f.....", "f....."}, render(consts.Coord[consts.Chunk]{X: 0, Y: 0}))
	assert.Equal(t, []string{"....ff", "......", "......", "......", "......"}, render(consts.Coord[consts.Chunk]{X: 1, Y: 0}))
	assert.Equal(t, []string{"......", "......", "......", "......", "....ff"}, render(consts.Coord[consts.Chunk]{X: 1, Y: 1}))
	assert.Equal(t, []string{".....f", ".....f", ".....f", ".....f", "ffffff"}, render(consts.Coord[consts.Chunk]{X: 2, Y: 1}))
}
