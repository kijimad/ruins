package uicore_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// TestGolden_UISpecPanel は widgets/uicore で組んだスペックパネルの実ピクセルを golden で固定する。
// EbitenCanvas で実フォント描画し、testdata/TestGolden_UISpecPanel.png と比較する。
// フェイスはプールから借りて独立させるので、ロック無し・t.Parallel で回せる。
func TestGolden_UISpecPanel(t *testing.T) {
	t.Parallel()
	res := borrowRes()
	defer facePool.Put(res)

	const w, h = 240, 160
	screen := ebiten.NewImage(w, h)
	panel := buildRealPanel(*res)
	panel.Layout(image.Rect(0, 0, w, h))
	panel.Draw(uicore.NewEbitenCanvas(screen))

	vrt.AssertFrameGolden(t, "TestGolden_UISpecPanel", screen)
}
