package interior

import (
	"sync"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

var (
	testRawsCache rawschema.Raws
	testRawsOnce  sync.Once
)

// testRaws は raw.toml をテスト用に一度だけロードして返す。本番は world.Resources.RawMaster を渡すが、
// テストは資材から直接ロードして全テストで共有する。読み込み失敗はテストの前提崩れなので panic。
func testRaws() rawschema.Raws {
	testRawsOnce.Do(func() {
		r, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
		if err != nil {
			panic("interior test: load raw: " + err.Error())
		}
		testRawsCache = r
	})
	return testRawsCache
}

// facilityIDs は raw.toml の全施設 id を返す。
func facilityIDs() []string {
	fs := raw.PtrSlice(testRaws().Facilities)
	ids := make([]string, len(fs))
	for i, f := range fs {
		ids[i] = f.Id
	}
	return ids
}

// testProfileTable は raw.toml の interiorProfile から組んだ抽選表を返す
func testProfileTable() profileTable {
	t, err := newProfileTable(testRaws())
	if err != nil {
		panic("interior test: profile table: " + err.Error())
	}
	return t
}
