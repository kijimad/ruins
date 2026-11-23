package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ringsLayers と rowsLayers は、今の町と研究所の敷地と同じ層
var (
	ringsLayers = []rawschema.DistrictLayer{
		{Zones: []rawschema.ZoneId{"residential", "industrial"}},
		{Zones: []rawschema.ZoneId{"downtown"}},
	}
	rowsLayers = []rawschema.DistrictLayer{
		{Zones: []rawschema.ZoneId{"campus_gate"}, Rows: new(1)},
		{Zones: []rawschema.ZoneId{"campus_inner"}},
		{Zones: []rawschema.ZoneId{"campus_quarantine"}, Rows: new(1)},
	}
)

// farSpawnRow は墜落現場の開始チャンクがどの区域にも当たらない遠い行。墜落現場と関係ない区域のテストで使う
const farSpawnRow consts.Chunk = -1 << 20

func TestDistrict_regionIn_大きさの候補から選び自分のリージョンに収める(t *testing.T) {
	t.Parallel()

	d := districtsFrom(rawschema.Raws{Districts: &[]rawschema.District{{
		Id: "town", Spacing: 7, Separation: 2, Pattern: rawschema.DistrictPatternRings, Layers: ringsLayers,
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
		Id: "town", Spacing: 7, Separation: 2, Pattern: rawschema.DistrictPatternRings, Layers: ringsLayers,
		Sizes: []rawschema.DistrictSize{{W: 3, H: 3, Weight: 1}},
	}}}
	const cols consts.Chunk = 7
	want := districtsFrom(raws)[0].regionIn(42, 2, cols)

	for y := consts.Chunk(14); y < 21; y++ {
		for x := range cols {
			c := consts.Coord[consts.Chunk]{X: x, Y: y}
			r, ok := districtAt(raws, 42, c, cols, farSpawnRow)
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

func TestDistrictAt_先の種類の区域と重なるか接する後の種類の区域は丸ごと出さない(t *testing.T) {
	t.Parallel()

	// 研究所の敷地は帯の全幅で縦の範囲が広く、町と必ず重なる
	raws := rawschema.Raws{Districts: &[]rawschema.District{
		{Id: "lab_campus", Spacing: 9, Separation: 1, Pattern: rawschema.DistrictPatternRows, Layers: rowsLayers, Fence: new(rawschema.EntityID("fence")), Sizes: []rawschema.DistrictSize{{W: 7, H: 8, Weight: 1}}},
		{Id: "town", Spacing: 3, Separation: 1, Pattern: rawschema.DistrictPatternRings, Layers: ringsLayers, Sizes: []rawschema.DistrictSize{{W: 2, H: 2, Weight: 1}}},
	}}
	const cols consts.Chunk = 7
	ds := districtsFrom(raws)

	for seed := range uint64(30) {
		campus := ds[0].regionIn(seed, 0, cols)
		for y := range consts.Chunk(9) {
			for x := range cols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				r, ok := districtAt(raws, seed, c, cols, farSpawnRow)
				if campus.contains(c) {
					require.True(t, ok)
					assert.Equal(t, "lab_campus", r.kind.Id, "seed=%d %v", seed, c)
					continue
				}
				town := ds[1].regionIn(seed, consts.FloorDiv(y, 3), cols)
				assert.Equal(t, town.contains(c) && !town.near(campus), ok, "seed=%d %v", seed, c)
			}
		}
	}
}

func TestDistricts_全ての大きさは地上の列数より狭い(t *testing.T) {
	t.Parallel()

	// 区域が帯の全幅を塞ぐと、区域の外がひと続きでなくなり、道がその先の行き先へ届かない
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	_, _, cols, _ := dungeon.DungeonOverworld.BandShape()
	for _, d := range raw.PtrSlice(raws.Districts) {
		for _, size := range d.Sizes {
			assert.Less(t, consts.Chunk(size.W), cols, "%s の幅 %d", d.Id, size.W)
		}
	}
}

func TestDistricts_seed100個の全てで通信施設が起点から北へ70チャンク以内に建つ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	cat := zoneCatalogFrom(raws)
	const within consts.Chunk = 70
	_, _, cols, rows := dungeon.DungeonOverworld.BandShape()
	spawnRow := gc.SeamlessBand{Rows: rows}.SpawnChunkY()
	for seed := range uint64(100) {
		found := false
		for depth := consts.Chunk(0); depth <= within && !found; depth++ {
			for x := range cols {
				c := consts.Coord[consts.Chunk]{X: x, Y: spawnRow - depth}
				if r, ok := districtAt(raws, seed, c, cols, spawnRow); ok && urbanFacilityAt(raws, cat, r, c, spawnRow) == "comms" {
					found = true
					break
				}
			}
		}
		assert.True(t, found, "seed=%d", seed)
	}
}
