package overworld

import (
	"bytes"
	"image/png"
	"math"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lakeSeed は開始の行の近くに湖の区域が出る seed。区域の左上は (5, -12)
const lakeSeed = 12

func TestDistrictRegion_waterAt_湖の区域は中心の層だけに湖を描く(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)
	require.Equal(t, "lakeside", r.kind.Id)

	got := make([]string, 0, r.h)
	for y := range r.h {
		line := ""
		for x := range r.w {
			if r.waterAt(consts.Coord[consts.Chunk]{X: x, Y: y}) != nil {
				line += "~"
			} else {
				line += "#"
			}
		}
		got = append(got, line)
	}
	assert.Equal(t, []string{"#######", "#~~~~~#", "#~~~~~#", "#~~~~~#", "#~~~~~#", "#~~~~~#", "#######"}, got)
	assert.Equal(t, "lakeside_water", urbanFacilityAt(raws, zoneCatalogFrom(raws), r, consts.Coord[consts.Chunk]{X: 8, Y: -9}, 0))
}

func TestRenderLakeChunk_楕円の輪郭の内に水面を置き外は陸に残す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	gen := NewChunkGen(world, lakeSeed, 24, 24, 15, mapplanner.PlannerTypeOverworldField)
	center := consts.Coord[consts.Chunk]{X: 8, Y: -9}
	corner := consts.Coord[consts.Chunk]{X: 6, Y: -11}
	require.NoError(t, gen(center, 0, 0))
	require.NoError(t, gen(corner, 24, 0))
	tiles := tileEntitiesInRange(world, 0, 48)
	at := func(x, y consts.Tile) string {
		return tileIDAt(world, tiles, consts.Coord[consts.Tile]{X: x, Y: y})
	}

	got := map[string]string{
		"真ん中":      at(12, 12),
		"角のチャンクの隅": at(24, 0),
	}
	assert.Equal(t, map[string]string{
		"真ん中":      "water_deep",
		"角のチャンクの隅": "dirt_tundra",
	}, got)
}

func TestGolden_Lakeside(t *testing.T) {
	t.Parallel()

	img, err := RenderNorthward(testutil.InitTestWorld(t), lakeSeed, 20, 0, 4)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	goldie.New(t, goldie.WithNameSuffix(".png")).Assert(t, t.Name(), buf.Bytes())
}

func TestDistrictRegion_builds_外の層は道の来る南の辺の中央に近いチャンクだけに施設を建てる(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)

	var got []consts.Coord[consts.Chunk]
	for y := range r.h {
		for x := range r.w {
			if l := (consts.Coord[consts.Chunk]{X: x, Y: y}); r.builds(l) {
				got = append(got, l)
			}
		}
	}
	require.Len(t, got, 2)
	assert.Contains(t, got, consts.Coord[consts.Chunk]{X: 3, Y: 6})
	assert.Equal(t, consts.Chunk(6), got[0].Y)
	assert.Equal(t, consts.Chunk(6), got[1].Y)
	assert.Empty(t, urbanFacilityAt(raws, zoneCatalogFrom(raws), r, consts.Coord[consts.Chunk]{X: 5, Y: -12}, 0), "施設を建てない角は施設を持たない")
}

func TestLakeShape_施設を建てたチャンクの正面から浜を越えて深い水まで桟橋を伸ばす(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)
	shape, ok, err := r.lakeShapeOf(24, 24)
	require.NoError(t, err)
	require.True(t, ok)

	c := r.anchor.Add(consts.Coord[consts.Chunk]{X: 3, Y: 6})
	x := c.X.Tiles(24) + 12
	var line []string
	for y := c.Y.Tiles(24); y > c.Y.Tiles(24)-r.h.Tiles(24); y-- {
		p := consts.Coord[consts.Tile]{X: x, Y: y}
		if shape.depth(p) >= float64(shape.water.BeachWidth+shape.water.ShallowWidth+shape.water.PierLength)+1 {
			break
		}
		if tile, ok := shape.tileAt(p); ok {
			line = append(line, tile)
		}
	}
	planks := slices.Index(line, "water_deep")
	require.Positive(t, planks)
	want := append(slices.Repeat([]string{"pier_plank"}, planks), slices.Repeat([]string{"water_deep"}, len(line)-planks)...)
	assert.Equal(t, want, line)
}

func TestLakeShapeOf_揺れと浜を引くと楕円が残らない層はエラーにする(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)
	kind := *r.kind
	kind.Layers = slices.Clone(kind.Layers)
	water := *kind.Layers[1].Water
	water.ShoreWobble = 5 * 24
	kind.Layers[1].Water = &water
	r.kind = &kind

	_, _, err := r.lakeShapeOf(24, 24)
	require.ErrorIs(t, err, errLakeTooSmall)
}

func TestGateColumns_門の位置が自然の地面なら真ん中の列を門にする(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)

	north, south := gateColumns(r, lotsOf(raws, zoneCatalogFrom(raws), r, 0))
	assert.Equal(t, [2]consts.Chunk{r.anchor.X + 3, r.anchor.X + 3}, [2]consts.Chunk{north, south})
}

func TestScatterCatalogForChunk_湖の区域は施設を建てないチャンクにだけ層の散布を撒く(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)

	got := map[string]outdoorZone{}
	for name, l := range map[string]consts.Coord[consts.Chunk]{"角の岸": {X: 0, Y: 0}, "湖": {X: 3, Y: 3}, "施設": {X: 3, Y: 6}} {
		cat, ok, err := scatterCatalogForChunk(raws, lakeSeed, r.anchor.Add(l), 15, 0)
		require.NoError(t, err)
		if ok {
			got[name] = cat.Zone
		}
	}
	assert.Equal(t, map[string]outdoorZone{"角の岸": "lakeshore", "湖": "lakeshore"}, got)
}

func TestLakeShape_桟橋の板は全て湖の輪郭の内側に張る(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)
	shape, ok, err := r.lakeShapeOf(24, 24)
	require.NoError(t, err)
	require.True(t, ok)

	require.NotEmpty(t, shape.piers)
	var onLand []consts.Coord[consts.Tile]
	for p := range shape.piers {
		if shape.depth(p) < 0 {
			onLand = append(onLand, p)
		}
	}
	assert.Equal(t, []consts.Coord[consts.Tile](nil), onLand)
}

func TestLayPier_湖の細い所で深さが浅くなり始めたら対岸まで張らずに止める(t *testing.T) {
	t.Parallel()
	// 中心を挟んで深さの等しい2マスまで張る
	s := lakeShape{
		water: &rawschema.DistrictWater{BeachWidth: 1, ShallowWidth: 1, PierLength: 100, ShoreCell: 8},
		cx:    10, cy: 10, rx: 5, ry: 5,
		piers: map[consts.Coord[consts.Tile]]bool{},
	}
	s.layPier(consts.Coord[consts.Tile]{X: 10, Y: 20}, consts.Coord[consts.Tile]{Y: -1}, 40)

	var ys []consts.Tile
	for p := range s.piers {
		if p.X == 10 {
			ys = append(ys, p.Y)
		}
	}
	slices.Sort(ys)
	assert.Equal(t, []consts.Tile{9, 10, 11, 12, 13, 14}, ys)
}

func TestLayPier_輪郭の揺れで深さが浅瀬より手前で減っても深い水まで張る(t *testing.T) {
	t.Parallel()
	s := lakeShape{
		water: &rawschema.DistrictWater{BeachWidth: 2, ShallowWidth: 3, PierLength: 6, ShoreWobble: 10, ShoreCell: 24},
		cx:    0, cy: 0, rx: 40, ry: 40, seed: 5,
		piers: map[consts.Coord[consts.Tile]]bool{},
	}
	dips := 0
	prev := math.Inf(-1)
	for y := consts.Tile(60); y > 0; y-- {
		d := s.depth(consts.Coord[consts.Tile]{Y: y})
		if d >= 0 && d < float64(s.water.BeachWidth+s.water.ShallowWidth) && d < prev {
			dips++
		}
		prev = d
	}
	require.Positive(t, dips, "前提: 浅瀬より手前で揺れを含む深さが減る")

	s.layPier(consts.Coord[consts.Tile]{Y: 60}, consts.Coord[consts.Tile]{Y: -1}, 120)

	deepest := math.Inf(-1)
	for p := range s.piers {
		if p.X == 0 {
			deepest = max(deepest, s.depth(p))
		}
	}
	assert.GreaterOrEqual(t, deepest, float64(s.water.BeachWidth+s.water.ShallowWidth))
}

func TestLayPier_輪郭の揺れで水から陸へ出たらその先の水へ板を飛ばさない(t *testing.T) {
	t.Parallel()
	s := lakeShape{
		water: &rawschema.DistrictWater{BeachWidth: 2, ShallowWidth: 3, PierLength: 6, ShoreWobble: 10, ShoreCell: 8},
		cx:    0, cy: 0, rx: 40, ry: 40, seed: 4,
		piers: map[consts.Coord[consts.Tile]]bool{},
	}
	var crossings []bool
	for y := consts.Tile(60); y > 0; y-- {
		d := s.depth(consts.Coord[consts.Tile]{Y: y})
		if d >= float64(s.water.BeachWidth+s.water.ShallowWidth+s.water.PierLength) {
			break
		}
		if wet := d >= 0; len(crossings) == 0 || crossings[len(crossings)-1] != wet {
			crossings = append(crossings, wet)
		}
	}
	require.Equal(t, []bool{false, true, false, true}, crossings, "前提: 水に入った後に一度陸へ出て、また水に入る")

	s.layPier(consts.Coord[consts.Tile]{Y: 60}, consts.Coord[consts.Tile]{Y: -1}, 120)

	var ys []consts.Tile
	for p := range s.piers {
		if p.X == 0 {
			ys = append(ys, p.Y)
		}
	}
	slices.Sort(ys)
	require.NotEmpty(t, ys)
	assert.Equal(t, ys[len(ys)-1]-ys[0]+1, consts.Tile(len(ys)), "板が途切れず1本につながる")
}

func TestChunkPlace_湖の区域は施設を建てない角を荒れ地に湖を湖の記号に南の辺の中央を水力発電所にする(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	anchor := consts.Coord[consts.Chunk]{X: 5, Y: -12}
	got := map[string]rune{}
	for name, l := range map[string]consts.Coord[consts.Chunk]{"角": {}, "湖": {X: 3, Y: 3}, "施設": {X: 3, Y: 6}} {
		got[name] = chunkPlace(raws, cat, lakeSeed, anchor.Add(l), 15, 0)
	}
	assert.Equal(t, map[string]rune{"角": '.', "湖": 'w', "施設": 'H'}, got)
}

func TestQuarterTurnsToward_反転の有無によらず北の辺を向きの側へ回す(t *testing.T) {
	t.Parallel()
	cells := [][]maptemplate.MapCell{{{Terrain: "n"}, {Terrain: "n"}, {Terrain: "n"}}, {{}, {}, {}}, {{}, {}, {}}}
	for _, mirror := range []bool{false, true} {
		got := map[consts.Coord[consts.Chunk]][][2]int{}
		for _, d := range []consts.Coord[consts.Chunk]{{Y: -1}, {X: -1}, {Y: 1}, {X: 1}} {
			for y, row := range maptemplate.TransformCells(cells, quarterTurnsToward(d), mirror) {
				for x, c := range row {
					if c.Terrain == "n" {
						got[d] = append(got[d], [2]int{x, y})
					}
				}
			}
		}
		assert.Equal(t, map[consts.Coord[consts.Chunk]][][2]int{
			{Y: -1}: {{0, 0}, {1, 0}, {2, 0}},
			{X: -1}: {{0, 0}, {0, 1}, {0, 2}},
			{Y: 1}:  {{0, 2}, {1, 2}, {2, 2}},
			{X: 1}:  {{2, 0}, {2, 1}, {2, 2}},
		}, got, "mirror=%v", mirror)
	}
}

func TestLandmarkLot_真ん中に建てられなければ同じ大きさで真ん中に一番近い区画に置く(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)
	f := rawschema.Facility{Id: "wide", Zones: []rawschema.FacilityZone{{Zone: "lake_shore"}}, Footprints: &[]rawschema.FacilityFootprint{{W: 2, H: 1, Weight: 1}}}
	southRow := func(x, y, w, h consts.Chunk, _ rawschema.ZoneId) bool {
		return y == 6 && h == 1 && x >= 0 && x+w <= r.w
	}

	anchor, w, h, ok := landmarkLot(r, f, southRow)
	require.True(t, ok)
	assert.Equal(t, []consts.Chunk{2, 6, 2, 1}, []consts.Chunk{anchor.X, anchor.Y, w, h})
}

func TestLotsOf_湖の区域は南の辺の中央に水力発電所を置き施設を全て湖へ向ける(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)

	type lotView struct {
		facility   string
		anchor     consts.Coord[consts.Chunk]
		facesWater bool
		turns      int
	}
	lots := lotsOf(raws, zoneCatalogFrom(raws), r, 0)
	got := make([]lotView, 0, len(lots))
	for _, l := range lots {
		got = append(got, lotView{l.facility, l.anchor, l.facesWater, l.turns})
	}
	assert.Equal(t, []lotView{
		{"hydro_plant", consts.Coord[consts.Chunk]{X: 3, Y: 6}, true, 0},
		{"fishing_hut", consts.Coord[consts.Chunk]{X: 4, Y: 6}, true, 0},
	}, got)
}

func TestRoadAt_道は湖の区域に入らず南の門の前まで来る(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)

	var inside []consts.Coord[consts.Chunk]
	for y := range r.h {
		for x := range r.w {
			c := r.anchor.Add(consts.Coord[consts.Chunk]{X: x, Y: y})
			if _, _, ok := roadAt(raws, lakeSeed, c, 15, 0); ok {
				inside = append(inside, c)
			}
		}
	}
	assert.Equal(t, []consts.Coord[consts.Chunk](nil), inside)

	_, south := gateColumns(r, lotsOf(raws, zoneCatalogFrom(raws), r, 0))
	ports, _, ok := roadAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: south, Y: r.anchor.Y + r.h}, 15, 0)
	require.True(t, ok)
	assert.Contains(t, ports, roadPort{dir: RoadN, lane: laneGate})
}

func TestWaterSide_湖が2方向に接するなら北南西東の順に先のほうを返す(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 6, Y: -11}, 15, 0)
	require.True(t, ok)

	got := map[consts.Coord[consts.Chunk]]consts.Coord[consts.Chunk]{}
	for _, l := range []consts.Coord[consts.Chunk]{{X: 3, Y: 6}, {X: 3, Y: 0}, {X: 0, Y: 3}, {X: 6, Y: 3}} {
		d, ok := r.waterSide(l)
		require.True(t, ok)
		got[l] = d
	}
	assert.Equal(t, map[consts.Coord[consts.Chunk]]consts.Coord[consts.Chunk]{
		{X: 3, Y: 6}: {Y: -1},
		{X: 3, Y: 0}: {Y: 1},
		{X: 0, Y: 3}: {X: 1},
		{X: 6, Y: 3}: {X: -1},
	}, got)

	// 湖の中のチャンクは北と西の両方が湖になる。北を先に見る
	inner := consts.Coord[consts.Chunk]{X: 2, Y: 2}
	require.NotNil(t, r.waterAt(inner.Add(consts.Coord[consts.Chunk]{Y: -1})))
	require.NotNil(t, r.waterAt(inner.Add(consts.Coord[consts.Chunk]{X: -1})))
	d, ok := r.waterSide(inner)
	require.True(t, ok)
	assert.Equal(t, consts.Coord[consts.Chunk]{Y: -1}, d)
}
