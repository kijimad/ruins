package replay_test

import (
	"os"
	"testing"

	"github.com/kijimaD/ruins/internal/vrt"
)

// TestMain は ebiten ゲームループの中で replay パッケージのテストを走らせる。PlayScenario は
// フレームを描いて画素を読むため ebiten の描画コンテキストが要り、画像操作を vrt.RunOnGameThread で
// ゲームループのスレッドへ寄せる。ループが無いと RunOnGameThread が drain されずデッドロックする。
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}
