package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistrict_regionIn_大きさの候補から選び自分のリージョンに収める(t *testing.T) {
	t.Parallel()

	d := districtsFrom(rawschema.Raws{Districts: &[]rawschema.District{{
		Id: "town", Spacing: 7, Separation: 2, Layout: rawschema.DistrictLayoutTown,
		Sizes: []rawschema.DistrictSize{{W: 2, H: 2, Weight: 1}, {W: 9, H: 5, Weight: 1}},
	}}})[0]
	const cols consts.Chunk = 7

	seen := map[[2]consts.Chunk]bool{}
	for seed := range uint64(300) {
		for ry := range consts.Chunk(5) {
			r := d.regionIn(seed, ry, cols)
			seen[[2]consts.Chunk{r.w, r.h}] = true
			assert.GreaterOrEqual(t, r.anchor.Y, ry*7)
			assert.LessOrEqual(t, r.anchor.Y+r.h, ry*7+5, "区域は次のリージョンとの空き separation を侵さない")
			assert.GreaterOrEqual(t, r.anchor.X, consts.Chunk(0))
			assert.LessOrEqual(t, r.anchor.X+r.w, cols)
		}
	}
	assert.Equal(t, map[[2]consts.Chunk]bool{{2, 2}: true, {7, 5}: true}, seen, "帯より広い候補は帯の全幅に詰める")
}

func TestDistrictAt_チャンクを含む区域を返す(t *testing.T) {
	t.Parallel()

	raws := rawschema.Raws{Districts: &[]rawschema.District{{
		Id: "town", Spacing: 7, Separation: 2, Layout: rawschema.DistrictLayoutTown,
		Sizes: []rawschema.DistrictSize{{W: 3, H: 3, Weight: 1}},
	}}}
	const cols consts.Chunk = 7
	want := districtsFrom(raws)[0].regionIn(42, 2, cols)

	for y := consts.Chunk(14); y < 21; y++ {
		for x := range cols {
			c := consts.Coord[consts.Chunk]{X: x, Y: y}
			r, ok := districtAt(raws, 42, c, cols)
			require.Equal(t, want.contains(c), ok, "%v", c)
			if ok {
				assert.Equal(t, want, r)
			}
		}
	}
}

func TestDistrictRegion_distance(t *testing.T) {
	t.Parallel()

	r := districtRegion{anchor: consts.Coord[consts.Chunk]{X: 2, Y: 3}, w: 2, h: 3}
	for _, tt := range []struct {
		c    consts.Coord[consts.Chunk]
		want consts.Chunk
	}{
		{consts.Coord[consts.Chunk]{X: 3, Y: 5}, 0},
		{consts.Coord[consts.Chunk]{X: 1, Y: 4}, 1},
		{consts.Coord[consts.Chunk]{X: 6, Y: 4}, 3},
		{consts.Coord[consts.Chunk]{X: 0, Y: 0}, 3},
		{consts.Coord[consts.Chunk]{X: 3, Y: 8}, 3},
	} {
		assert.Equal(t, tt.want, r.distance(tt.c), "%v", tt.c)
	}
}

func TestDistrictAt_先の種類の区域と重なる後の種類の区域は丸ごと出さない(t *testing.T) {
	t.Parallel()

	// 研究所の敷地は帯の全幅で縦の範囲が広く、町と必ず重なる
	raws := rawschema.Raws{Districts: &[]rawschema.District{
		{Id: "lab_campus", Spacing: 9, Separation: 1, Layout: rawschema.DistrictLayoutCampus, Sizes: []rawschema.DistrictSize{{W: 7, H: 8, Weight: 1}}},
		{Id: "town", Spacing: 3, Separation: 1, Layout: rawschema.DistrictLayoutTown, Sizes: []rawschema.DistrictSize{{W: 2, H: 2, Weight: 1}}},
	}}
	const cols consts.Chunk = 7
	ds := districtsFrom(raws)

	for seed := range uint64(30) {
		campus := ds[0].regionIn(seed, 0, cols)
		for y := range consts.Chunk(9) {
			for x := range cols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				r, ok := districtAt(raws, seed, c, cols)
				if campus.contains(c) {
					require.True(t, ok)
					assert.Equal(t, rawschema.DistrictLayoutCampus, r.layout, "seed=%d %v", seed, c)
					continue
				}
				town := ds[1].regionIn(seed, consts.FloorDiv(y, 3), cols)
				assert.Equal(t, town.contains(c) && !town.overlaps(campus), ok, "seed=%d %v", seed, c)
			}
		}
	}
}
