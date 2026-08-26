package ui_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/vrt"
)

// TestGolden_UISpecPanel は internal/ui で組んだスペックパネルの実ピクセルを golden で固定する。
// EbitenCanvas で実フォント描画し、testdata/TestGolden_UISpecPanel.png と比較する。
// フェイスはプールから借りて独立させるので、ロック無し・t.Parallel で回せる。
// これで自前 UI の実画面が VRT の golden 流儀に載ることを確かめ、画面移行のパターンを確立する。
func TestGolden_UISpecPanel(t *testing.T) {
	t.Parallel()
	res := borrowRes()
	defer facePool.Put(res)

	const w, h = 240, 160
	screen := ebiten.NewImage(w, h)
	u := ui.New(buildRealPanel(*res))
	u.Layout(image.Rect(0, 0, w, h))
	u.Draw(ui.NewEbitenCanvas(screen))

	vrt.AssertFrameGolden(t, "TestGolden_UISpecPanel", screen)
}
