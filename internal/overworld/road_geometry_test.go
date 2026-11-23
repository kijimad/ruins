package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoadsideOf(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    consts.Coord[consts.Chunk]
		cols consts.Chunk
		want consts.Coord[consts.Chunk]
		ok   bool
	}{
		{"西隣を通る", consts.Coord[consts.Chunk]{X: 3, Y: 5}, 7, consts.Coord[consts.Chunk]{X: 2, Y: 5}, true},
		{"西端の列は東隣を通る", consts.Coord[consts.Chunk]{X: 0, Y: 5}, 7, consts.Coord[consts.Chunk]{X: 1, Y: 5}, true},
		{"列が1つなら脇が無い", consts.Coord[consts.Chunk]{X: 0, Y: 5}, 1, consts.Coord[consts.Chunk]{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := roadsideOf(tt.s, tt.cols)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLinkChunks(t *testing.T) {
	t.Parallel()
	type c = consts.Coord[consts.Chunk]
	tests := []struct {
		name string
		a, b c
		cols consts.Chunk
		want []c
	}{
		{"道ばたから縦に進み、間の行で折れて相手の道ばたへ進む", c{X: 3, Y: 0}, c{X: 5, Y: 6}, 7,
			[]c{{X: 2, Y: 0}, {X: 2, Y: 1}, {X: 2, Y: 2}, {X: 2, Y: 3}, {X: 3, Y: 3}, {X: 4, Y: 3}, {X: 4, Y: 4}, {X: 4, Y: 5}, {X: 4, Y: 6}}},
		{"間に行が無ければ北の外側の行で折れる", c{X: 2, Y: 0}, c{X: 4, Y: 1}, 7,
			[]c{{X: 1, Y: 0}, {X: 1, Y: -1}, {X: 2, Y: -1}, {X: 3, Y: -1}, {X: 3, Y: 0}, {X: 3, Y: 1}}},
		{"どの行で折れても集落を通るなら道を作らない", c{X: 0, Y: 0}, c{X: 1, Y: 1}, 2, nil},
		{"列が1つなら道を作らない", c{X: 0, Y: 0}, c{X: 0, Y: 5}, 1, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, linkChunks(tt.a, tt.b, tt.cols))
		})
	}
}

func TestRoadChunkSegs_中心の四角を含めて縁まで伸ばす(t *testing.T) {
	t.Parallel()
	// チャンク (1,2) の中心は (36,60)。幅4の帯は中心線の -2 から +1 なので、中心の四角は 34 から 37
	got := roadChunkSegs(consts.Coord[consts.Chunk]{X: 1, Y: 2}, RoadN|RoadE, 4, 24, 24)

	assert.Equal(t, []roadSeg{
		{orient: orientVertical, fixed: 36, lo: 48, hi: 61},
		{orient: orientHorizontal, fixed: 60, lo: 34, hi: 47},
	}, got)
}

func TestChunkTypeAt_道は点在ランドマークより優先する(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	const cols consts.Chunk = 7
	found := 0
	for seed := range uint64(50) {
		for y := range consts.Chunk(30) {
			for x := range cols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if !placementOf(raws, rawschema.PointFeatureKindLandmark).At(seed, c, cols) || !onRoad(raws, seed, c, cols) {
					continue
				}
				if _, ok := districtAt(raws, seed, c, cols, 100); ok || placementOf(raws, rawschema.PointFeatureKindSettlement).At(seed, c, cols) {
					continue
				}
				found++
				assert.Equal(t, chunkRoad, chunkTypeAt(raws, seed, c, cols, 100), "seed=%d %v", seed, c)
			}
		}
	}
	require.Positive(t, found, "前提: 道とランドマークの当選が重なるチャンクがある")
}
