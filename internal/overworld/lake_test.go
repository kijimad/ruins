package overworld

import (
	"bytes"
	"image/png"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
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
	for y := c.Y.Tiles(24); ; y-- {
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
	want := append(slices.Repeat([]string{"plank_brown"}, planks), slices.Repeat([]string{"water_deep"}, len(line)-planks)...)
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
