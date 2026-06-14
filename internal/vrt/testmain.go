package vrt

import (
	"fmt"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// testHostGame はテスト実行中にebitenグラフィックスコンテキストを維持するゲーム
type testHostGame struct {
	testFunc func() int
	started  bool
}

func (g *testHostGame) Update() error {
	if !g.started {
		g.started = true
		go func() {
			// ゲームループが回っている状態で、テストは別goroutineとして実行する
			code := g.testFunc()
			// テスト完了後、ebitenのシャットダウン処理でEGL/X11ドライバがsegfaultすることがあるため、
			// テスト結果が確定した時点で即座にプロセスを終了する
			os.Exit(code)
		}()
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
	if err := ebiten.RunGame(g); err != nil {
		fmt.Fprintf(os.Stderr, "RunGame failed: %v\n", err)
		return 1
	}
	// テストが正常完了した場合、goroutine内でos.Exitが呼ばれるためここには到達しない
	return 0
}
