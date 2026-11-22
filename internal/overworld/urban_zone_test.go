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

func TestInCore_大きい区域ほど中心が広い(t *testing.T) {
	t.Parallel()

	downtowns := func(w, h consts.Chunk) []consts.Coord[consts.Chunk] {
		var out []consts.Coord[consts.Chunk]
		for ly := range h {
			for lx := range w {
				if inCore(lx, ly, w, h) {
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

func TestDistrictRegion_zoneAt_層の候補を区域のseedで1つ選ぶ(t *testing.T) {
	t.Parallel()

	town := &rawschema.District{Pattern: rawschema.DistrictPatternRings, Layers: ringsLayers}
	zones := func(seed uint64) []rawschema.ZoneId {
		r := districtRegion{kind: town, w: 3, h: 3, seed: seed}
		return []rawschema.ZoneId{r.zoneAt(consts.Coord[consts.Chunk]{X: 0, Y: 0}), r.zoneAt(consts.Coord[consts.Chunk]{X: 1, Y: 1})}
	}
	assert.Equal(t, []rawschema.ZoneId{"residential", "downtown"}, zones(0))
	assert.Equal(t, []rawschema.ZoneId{"industrial", "downtown"}, zones(1<<zoneChoiceShift))
}

// TestRollFacilityInZone_規模gateで専門施設はspan3以上でだけ出る は、都心の骨董品店・診療所が minSpan=3 なので
// span=2 の都心では出ず、span=3 で初めて混ざることを固定する。住宅地と産業区では span=2 でも出る
func TestRollFacilityInZone_規模gateで専門施設はspan3以上でだけ出る(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	small := map[string]bool{}
	big := map[string]bool{}
	for s := range uint64(300) {
		small[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), cat, "downtown", 2, 1, "")] = true
		big[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), cat, "downtown", 3, 1, "")] = true
	}
	for _, f := range []string{"antique", "clinic"} {
		assert.Falsef(t, small[f], "span=2 の都心に minSpan=3 の %s は出ない", f)
	}
	assert.True(t, big["antique"] || big["clinic"],
		"span=3 の都心では専門施設が混ざる")
}

func TestRollFacilityInZone_住宅地と産業区は一番小さい町でも珍しい施設が混ざる(t *testing.T) {
	t.Parallel()

	cat := zoneCatalogFrom(testutil.InitTestWorld(t).Resources.RawMaster)
	seen := func(z rawschema.ZoneId) []string {
		got := map[string]bool{}
		for s := range uint64(300) {
			got[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), cat, z, 2, 1, "")] = true
		}
		return slices.Sorted(maps.Keys(got))
	}
	assert.Equal(t, []string{"antique", "clinic", "gunshop", "house", "store"}, seen("residential"))
	assert.Equal(t, []string{"clinic", "depot", "gunshop", "house", "office"}, seen("industrial"))
}

func TestRollFacilityInZone_深さの下限に届く危険度でだけ出る(t *testing.T) {
	t.Parallel()

	cat := zoneCatalog{"downtown": {
		{kind: "store", weight: 1, minSpan: 1, minDanger: 1},
		{kind: "comms", weight: 1, minSpan: 1, minDanger: 3},
	}}
	seen := func(danger consts.Danger) []string {
		got := map[string]bool{}
		for s := range uint64(100) {
			got[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), cat, "downtown", 1, danger, "")] = true
		}
		return slices.Sorted(maps.Keys(got))
	}
	assert.Equal(t, []string{"store"}, seen(2))
	assert.Equal(t, []string{"comms", "store"}, seen(3))
}

func TestLotsOf_施設の深さは起点の行から測った区域の北の端の危険度で決まる(t *testing.T) {
	t.Parallel()

	town := &rawschema.District{Id: "town", Pattern: rawschema.DistrictPatternRings, Layers: []rawschema.DistrictLayer{
		{Zones: []rawschema.ZoneId{"residential"}},
		{Zones: []rawschema.ZoneId{"downtown"}},
	}}
	raws := rawschema.Raws{Facilities: &[]rawschema.Facility{
		{Id: "house", Zones: []rawschema.FacilityZone{{Zone: "residential", Weight: 1, MinSpan: 1}, {Zone: "downtown", Weight: 1, MinSpan: 1}}},
		{Id: "comms", Zones: []rawschema.FacilityZone{{Zone: "residential", Weight: 1000, MinSpan: 1, MinDanger: new(2)}}},
	}}
	cat := zoneCatalogFrom(raws)
	r := districtRegion{kind: town, anchor: consts.Coord[consts.Chunk]{X: 0, Y: -2}, w: 2, h: 2, seed: 1}
	facilities := func(spawnRow consts.Chunk) []string {
		got := map[string]bool{}
		for _, l := range lotsOf(raws, cat, r, spawnRow) {
			got[l.facility] = true
		}
		return slices.Sorted(maps.Keys(got))
	}
	// 危険度は北進3チャンクで1段上がる。区域の北の端は起点から2チャンク北なら1、3チャンク北なら2
	assert.Equal(t, []string{"house"}, facilities(0))
	assert.Equal(t, []string{"comms"}, facilities(1))
}

func TestRollFacilityInZone_実データの通信施設は危険度6から都心に混ざる(t *testing.T) {
	t.Parallel()

	cat := zoneCatalogFrom(testutil.InitTestWorld(t).Resources.RawMaster)
	seen := func(danger consts.Danger) []string {
		got := map[string]bool{}
		for s := range uint64(300) {
			got[rollFacilityInZone(rand.New(rand.NewPCG(s, 0)), cat, "downtown", 3, danger, "")] = true
		}
		return slices.Sorted(maps.Keys(got))
	}
	assert.Equal(t, slices.Sorted(slices.Values(append(seen(5), "comms"))), seen(6))
}

// TestRollFacilityInZone_決定的 は、同じ rng 状態から同じ施設が出て、必ず有効な種別を返すことを固定する。
func TestRollFacilityInZone_決定的(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	first := rollFacilityInZone(rand.New(rand.NewPCG(7, 0)), cat, "residential", 3, 1, "")
	assert.NotEmpty(t, first, "有効な施設種別を返す")
	for range 5 {
		got := rollFacilityInZone(rand.New(rand.NewPCG(7, 0)), cat, "residential", 3, 1, "")
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
				if r, ok := districtAt(raws, s, c, rows, farSpawnRow); ok {
					k := urbanFacilityAt(raws, cat, r, c, 0)
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

func TestRowsLayer_行の数を書いた層が南と北の端から並び残りを1つの層が占める(t *testing.T) {
	t.Parallel()

	layers := func(rows ...int) []rawschema.DistrictLayer {
		out := make([]rawschema.DistrictLayer, len(rows))
		for i, n := range rows {
			if n > 0 {
				out[i].Rows = new(n)
			}
		}
		return out
	}
	got := func(ls []rawschema.DistrictLayer, h consts.Chunk) []int {
		out := make([]int, h)
		for ly := range h {
			out[ly] = rowsLayer(ls, ly, h)
		}
		return out
	}
	assert.Equal(t, []int{2, 1, 1, 1, 1, 0}, got(layers(1, 0, 1), 6), "研究所の敷地。北の行から順に並べる")
	assert.Equal(t, []int{3, 3, 2, 2, 1, 0}, got(layers(1, 1, 0, 2), 6))
	assert.Equal(t, []int{1, 1, 0}, got(layers(0, 2), 3), "残りの層が南の端に来る")
}

func TestDistrictRegion_zoneAt_区域が割り振る地区はrawの地区の一覧に収まる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	for _, d := range districtsFrom(raws) {
		got := map[rawschema.ZoneId]bool{}
		for _, size := range d.row.Sizes {
			for seed := range uint64(4) {
				r := districtRegion{kind: d.row, w: consts.Chunk(size.W), h: consts.Chunk(size.H), seed: seed << zoneChoiceShift}
				for ly := range r.h {
					for lx := range r.w {
						got[r.zoneAt(consts.Coord[consts.Chunk]{X: lx, Y: ly})] = true
					}
				}
			}
		}
		assert.ElementsMatch(t, raw.DistrictZones(*d.row), slices.Collect(maps.Keys(got)), d.row.Id)
	}
}

func TestDistrictFence_外周を巡り門の列の南北に門を開ける(t *testing.T) {
	t.Parallel()

	const w, h consts.Tile = 6, 5
	r := districtRegion{w: 3, h: 2}
	render := func(l consts.Coord[consts.Chunk]) []string {
		rows := make([][]byte, h)
		for y := range rows {
			rows[y] = []byte("......")
		}
		for _, p := range districtFence(r, l, w, h, 1, 1) {
			rows[p.Y][p.X] = 'f'
		}
		out := make([]string, h)
		for y := range rows {
			out[y] = string(rows[y])
		}
		return out
	}
	// 街路の幅 urbanStreetW は4。門の列 lx=1 の北の端の行と南の端の行だけ、西の4マスを開ける
	assert.Equal(t, []string{"ffffff", "f.....", "f.....", "f.....", "f....."}, render(consts.Coord[consts.Chunk]{X: 0, Y: 0}))
	assert.Equal(t, []string{"....ff", "......", "......", "......", "......"}, render(consts.Coord[consts.Chunk]{X: 1, Y: 0}))
	assert.Equal(t, []string{"......", "......", "......", "......", "....ff"}, render(consts.Coord[consts.Chunk]{X: 1, Y: 1}))
	assert.Equal(t, []string{".....f", ".....f", ".....f", ".....f", "ffffff"}, render(consts.Coord[consts.Chunk]{X: 2, Y: 1}))
}

func TestStreetTerrain_植え込みは帯の両端で角は道(t *testing.T) {
	t.Parallel()

	render := func(s streetTiles) []string {
		out := make([]string, 0, 6)
		for y := range consts.Tile(6) {
			row := ""
			for x := range consts.Tile(6) {
				switch {
				case x >= urbanStreetW && y >= urbanStreetW:
					row += " "
				case streetTerrain(x, y, s) == "grass_green":
					row += ","
				default:
					row += "."
				}
			}
			out = append(out, row)
		}
		return out
	}
	assert.Equal(t, []string{
		"....,,",
		"......",
		"......",
		"....,,",
		",..,  ",
		",..,  ",
	}, render(streetTiles{street: consts.TileNameRoad, verge: "grass_green"}))
	assert.Equal(t, []string{
		"......",
		"......",
		"......",
		"......",
		"....  ",
		"....  ",
	}, render(streetTiles{street: consts.TileNameRoad}))
}

func TestStreetTilesOf_植え込みを省くと植え込みを置かない(t *testing.T) {
	t.Parallel()

	assert.Equal(t, streetTiles{street: "road"}, streetTilesOf(&rawschema.District{Street: "road"}))
	assert.Equal(t, streetTiles{street: "road", verge: "grass_green"}, streetTilesOf(&rawschema.District{Street: "road", Verge: new(rawschema.EntityID("grass_green"))}))
}
