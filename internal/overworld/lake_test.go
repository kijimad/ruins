package overworld

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lakeSeed は開始の行の近くに湖の区域が出る seed。地上の15列で、区域の左上は (5, -12)、水で満たすのは (6, -11) から 5x5
const lakeSeed = 12

func TestDistrictRegion_waterAt_湖の区域は中心だけを水で満たす(t *testing.T) {
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
	// 湖の真ん中のチャンクと、北西の角のチャンク
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
	// 南の辺の中央 (3, 6) と、同じ近さの隣のどちらか
	require.Len(t, got, 2)
	assert.Contains(t, got, consts.Coord[consts.Chunk]{X: 3, Y: 6})
	assert.Equal(t, consts.Chunk(6), got[0].Y)
	assert.Equal(t, consts.Chunk(6), got[1].Y)
	assert.Empty(t, urbanFacilityAt(raws, zoneCatalogFrom(raws), r, consts.Coord[consts.Chunk]{X: 5, Y: -12}, 0), "施設を建てない角は施設を持たない")
}
