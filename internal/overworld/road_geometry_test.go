package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestRoadSegments_水平辺は始点の行_垂直辺は終点の列でチャンクの中心を通る(t *testing.T) {
	t.Parallel()
	// chunkW=24, chunkH=16 と非対称にして軸の取り違えを検出する
	segs := roadSegments(consts.Coord[consts.Chunk]{X: 0, Y: 0}, consts.Coord[consts.Chunk]{X: 2, Y: 3}, 24, 16)

	assert.Equal(t, []roadSeg{
		{orient: orientHorizontal, fixed: 8, lo: 12, hi: 2*24 + 12},
		{orient: orientVertical, fixed: 2*24 + 12, lo: 8, hi: 3*16 + 8},
	}, segs)
}

func TestRoadSegments_端点は大小に依らず正規化される(t *testing.T) {
	t.Parallel()
	segs := roadSegments(consts.Coord[consts.Chunk]{X: 4, Y: 3}, consts.Coord[consts.Chunk]{X: 1, Y: 1}, 2, 2)

	assert.Equal(t, []roadSeg{
		{orient: orientHorizontal, fixed: 7, lo: 3, hi: 9},
		{orient: orientVertical, fixed: 3, lo: 3, hi: 7},
	}, segs)
}

func TestRoadPath_塀に囲まれた区域は南北の門を通る(t *testing.T) {
	t.Parallel()

	raws := rawschema.Raws{
		Districts: &[]rawschema.District{{
			Id: "lab_campus", Spacing: 20, Separation: 1, Layout: rawschema.DistrictLayoutCampus,
			Sizes: []rawschema.DistrictSize{{W: 3, H: 4, Weight: 1}},
		}},
		Facilities: &[]rawschema.Facility{
			{Id: "gate", Zones: []rawschema.FacilityZone{{Zone: rawschema.CampusGate, Weight: 1, MinSpan: 1}}},
			{Id: "inner", Zones: []rawschema.FacilityZone{{Zone: rawschema.CampusInner, Weight: 1, MinSpan: 1}}},
			{Id: "back", Zones: []rawschema.FacilityZone{{Zone: rawschema.CampusQuarantine, Weight: 1, MinSpan: 1}}},
		},
	}
	const cols consts.Chunk = 3
	const size consts.Tile = 10
	r := districtsFrom(raws)[0].regionIn(1, 0, cols)
	a := consts.Coord[consts.Chunk]{X: 0, Y: r.anchor.Y - 3}
	b := consts.Coord[consts.Chunk]{X: 2, Y: r.anchor.Y + r.h + 2}
	north, south := campusGateColumns(raws, r)
	lane := func(col consts.Chunk) consts.Tile { return col.Tiles(size) + urbanStreetW/2 }
	top, bottom := r.anchor.Y.Tiles(size), (r.anchor.Y+r.h).Tiles(size)-1
	before, after := chunkCenter(r.anchor.Y-1, size), chunkCenter(r.anchor.Y+r.h, size)
	x := chunkCenter(b.X, size)

	assert.Equal(t, []roadSeg{
		span(orientHorizontal, chunkCenter(a.Y, size), chunkCenter(a.X, size), x),
		span(orientVertical, x, chunkCenter(a.Y, size), before),
		span(orientHorizontal, before, x, lane(north)),
		span(orientVertical, lane(north), before, top),
		span(orientVertical, lane(south), bottom, after),
		span(orientHorizontal, after, lane(south), x),
		span(orientVertical, x, after, chunkCenter(b.Y, size)),
	}, roadPath(raws, 1, a, b, cols, size, size))
}
