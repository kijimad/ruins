package save

import (
	"os"
	"testing"

	"github.com/kijimaD/ruins/internal/vrt"
)

// TestMain はebitenグラフィックスコンテキスト内で全テストを実行する。
// testutil.InitTestWorld が UIResources をロードし ebiten の実行状態に依存するため、
// RunGame の外で走らせると glfw が実ディスプレイを開こうとして xvfb と競合し稀に失敗する。
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}
