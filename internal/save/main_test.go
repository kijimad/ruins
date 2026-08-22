package save

import (
	"os"
	"testing"

	"github.com/kijimaD/ruins/internal/vrt"
)

// TestMain はebitenグラフィックスコンテキスト内で全テストを実行する。
// UIResources のロードが ebiten の実行状態に依存するため必要
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}
