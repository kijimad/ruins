package interior

import (
	"sync"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
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

// facSpec は施設 id から FacilitySpec を testRaws 経由で解決するテストヘルパ。本番の overworld と同じく
// facilities 行から planner と isShop を引く。未登録 id は前提崩れなので panic。
func facSpec(id string) FacilitySpec {
	f, ok := raw.GetFacility(testRaws(), id)
	if !ok {
		panic("interior test: facility not found: " + id)
	}
	return FacilitySpec{ID: f.Id, Planner: f.Planner, IsShop: f.IsShop}
}
