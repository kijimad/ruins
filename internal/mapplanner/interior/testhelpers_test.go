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

// testRoomContents は施設の役割別 content をテスト用に map で返す。
func testRoomContents(t *testing.T, fac FacilityKind) map[roleName]Content {
	t.Helper()
	raws := testRaws()
	out := map[roleName]Content{}
	for _, fr := range raw.PtrSlice(raws.FacilityRooms) {
		if FacilityKind(fr.Facility) != fac {
			continue
		}
		for _, r := range raw.PtrSlice(fr.Rooms) {
			c, err := contentByID(raws, r.Content)
			require.NoError(t, err)
			out[roleName(r.Role)] = c
		}
	}
	return out
}

// diningTableStuff は椅子を四辺へ束ねた食卓の Stuff。衛星配置のテスト専用フィクスチャで、本番レシピは
// raw.toml が持つ。
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
