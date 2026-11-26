package overworld

import (
	"bytes"
	"image/png"
	"math"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lakeSeed は開始の行の近くに湖の区域が出る seed。区域の左上は (1, -12) で、水で満たすのは (2, -11) から 5x5
const lakeSeed = 12

func TestDistrictRegion_waterAt_湖の区域は中心だけを水で満たす(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	r, ok := districtAt(raws, lakeSeed, consts.Coord[consts.Chunk]{X: 2, Y: -11}, 9, 0)
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
	assert.Equal(t, "lakeside_water", urbanFacilityAt(raws, zoneCatalogFrom(raws), r, consts.Coord[consts.Chunk]{X: 4, Y: -9}, 0))
}

func TestLakeShore_landDistance(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		shore lakeShore
		x, y  consts.Tile
		want  float64
	}{
		{"北の辺の1行目", lakeShore{north: true}, 10, 0, 0},
		{"北の辺から3行", lakeShore{north: true}, 10, 3, 3},
		{"東の辺は右端からの距離", lakeShore{east: true}, 20, 5, 3},
		{"近い方の辺を取る", lakeShore{north: true, west: true}, 2, 5, 2},
		{"陸に面さない", lakeShore{}, 10, 10, math.Inf(1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tc.want, tc.shore.landDistance(tc.x, tc.y, 24, 24), 1e-9)
		})
	}
}

func TestRenderLakeChunk_陸に面する辺から浜と浅瀬と深い水を並べる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	gen := NewChunkGen(world, lakeSeed, 24, 24, 9, mapplanner.PlannerTypeOverworldField)
	// 湖の北西の角のチャンク。北と西が陸に面する
	c := consts.Coord[consts.Chunk]{X: 2, Y: -11}
	require.NoError(t, gen(c, 0, 0))
	tiles := tileEntitiesInRange(world, 0, 24)
	at := func(x, y consts.Tile) string {
		return tileIDAt(world, tiles, consts.Coord[consts.Tile]{X: x, Y: y})
	}

	got := map[string]string{
		"北の浜":   at(12, 0),
		"北の浜の内": at(12, 1),
		"西の浜":   at(0, 12),
		"北の浅瀬":  at(12, 2),
		"西の浅瀬":  at(2, 12),
		"真ん中":   at(12, 12),
		"東の端":   at(23, 12),
		"南の端":   at(12, 23),
	}
	assert.Equal(t, map[string]string{
		"北の浜":   "sand_slate",
		"北の浜の内": "sand_slate",
		"西の浜":   "sand_slate",
		"北の浅瀬":  "water_shallow",
		"西の浅瀬":  "water_shallow",
		"真ん中":   "water_deep",
		"東の端":   "water_deep",
		"南の端":   "water_deep",
	}, got)
}

func TestGolden_Lakeside(t *testing.T) {
	t.Parallel()

	img, err := RenderNorthward(testutil.InitTestWorld(t), lakeSeed, 17, 0, 4)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	goldie.New(t, goldie.WithNameSuffix(".png")).Assert(t, t.Name(), buf.Bytes())
}
