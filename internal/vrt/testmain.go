package vrt

import (
	"fmt"
	"os"
	"sync/atomic"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// testHostGame はテスト実行中にebitenグラフィックスコンテキストを維持するゲーム
type testHostGame struct {
	testFunc func() int
	exitCode int
	done     atomic.Bool
	started  bool
}

func (g *testHostGame) Update() error {
	if !g.started {
		g.started = true
		go func() {
			// ゲームループが回っている状態で、テストは別goroutineとして実行する
			g.exitCode = g.testFunc()
			g.done.Store(true)
		}()
	}
	if g.done.Load() {
		return ebiten.Termination
	}
	return nil
}

func (g *testHostGame) Draw(_ *ebiten.Image) {}

func (g *testHostGame) Layout(_, _ int) (int, int) {
	return 1, 1
}

// RunTestMain はTestMainから呼び出し、ebitenゲームループ内で全テストを実行する。
// これにより各テスト関数で ebiten.NewImage + ReadPixels が使える。
// gamepad初期化のEINTR回避のため、bwrapで/dev/inputを隠した環境で実行すること
func RunTestMain(m *testing.M) int {
	g := &testHostGame{testFunc: m.Run}
	// ebiten.RunGame()はメインスレッドをブロックする。
	// ebiten.Terminationを返すとRunGameはnilを返して正常終了する
	if err := ebiten.RunGame(g); err != nil {
		fmt.Fprintf(os.Stderr, "RunGame failed: %v\n", err)
		return 1
	}
	return g.exitCode
}
