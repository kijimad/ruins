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
