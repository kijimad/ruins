package interior

import (
	"sync"
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/require"
)

var (
	testRawsCache oapi.Raws
	testRawsOnce  sync.Once
)

// testRaws は raw.toml をテスト用に一度だけロードして返す。本番は world.Resources.RawMaster を渡すが、
// テストは資材から直接ロードして全テストで共有する。読み込み失敗はテストの前提崩れなので panic。
func testRaws() oapi.Raws {
	testRawsOnce.Do(func() {
		r, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
		if err != nil {
			panic("interior test: load raw: " + err.Error())
		}
		testRawsCache = r
	})
	return testRawsCache
}

// must* は生成系の error を require で潰し、happy path だけ見たいテストの記述を短く保つ包み。
func mustContent(t *testing.T, id string) Content {
	t.Helper()
	c, err := contentByID(testRaws(), id)
	require.NoError(t, err)
	return c
}

func mustFurnish(t *testing.T, seed uint64, footprint Rect, door Vec, facility FacilityKind) []Placed {
	t.Helper()
	placed, err := Furnish(testRaws(), seed, footprint, door, facility)
	require.NoError(t, err)
	return placed
}

func mustFurnishBuilding(t *testing.T, seed uint64, footprint Rect, door Vec, facility FacilityKind) (Site, []Placed) {
	t.Helper()
	site, placed, err := FurnishBuilding(testRaws(), seed, footprint, door, facility)
	require.NoError(t, err)
	return site, placed
}

func mustFurnishStages(t *testing.T, seed uint64, footprint Rect, door Vec, facility FacilityKind) (Site, []FurnishStage) {
	t.Helper()
	site, stages, err := FurnishStages(testRaws(), seed, footprint, door, facility)
	require.NoError(t, err)
	return site, stages
}

// testRoomContents は施設の役割別 content をテスト用に map で返す。旧 houseRoomContents 等の代わり。
func testRoomContents(t *testing.T, fac FacilityKind) map[roleName]Content {
	t.Helper()
	raws := testRaws()
	out := map[roleName]Content{}
	for _, fr := range raw.PtrSlice(raws.FacilityRooms) {
		if FacilityKind(fr.Facility) != fac {
			continue
		}
		for _, r := range raw.PtrSlice(fr.Rooms) {
			out[roleName(r.Role)] = mustContent(t, r.Content)
		}
	}
	return out
}

// diningTableStuff は椅子を四辺へ束ねた食卓の Stuff。衛星配置のテスト専用フィクスチャで、本番レシピは
// raw.toml が持つ。旧 fixtures.go の diningTable をテストへ移したもの。
func diningTableStuff(placement Placement) Stuff {
	chair := func(offs ...Vec) Satellite {
		return Satellite{Kind: KindFurniture, Ref: "chair", Offsets: offs}
	}
	return Stuff{
		Kind: KindFurniture, Ref: "table", Placement: placement, Amount: consts.Dice{Base: 1, Sides: 1},
		Satellites: []Satellite{
			chair(Vec{X: 0, Y: -1}, Vec{X: -1, Y: -1}, Vec{X: 1, Y: -1}),
			chair(Vec{X: 0, Y: 1}, Vec{X: -1, Y: 1}, Vec{X: 1, Y: 1}),
			chair(Vec{X: -1, Y: 0}, Vec{X: -1, Y: -1}, Vec{X: -1, Y: 1}),
			chair(Vec{X: 1, Y: 0}, Vec{X: 1, Y: -1}, Vec{X: 1, Y: 1}),
		},
	}
}
