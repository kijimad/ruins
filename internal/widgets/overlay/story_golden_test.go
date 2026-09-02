package overlay

import (
	"fmt"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// 詳細モーダルの部品カタログ。固定入力で実描画し、見た目を部品の粒度で golden に固定する。
// 先頭ページと最終ページで、性能行のページ送りと説明の出し分けを画で確かめる。

func storySpecRows() []entityspec.SpecRow {
	rows := make([]entityspec.SpecRow, 15)
	for i := range rows {
		rows[i] = entityspec.SpecRow{Label: fmt.Sprintf("Stat %02d", i), Value: fmt.Sprintf("%d", i*3)}
	}
	return rows
}

func drawDetailStory(t *testing.T, name string, page int) {
	t.Helper()
	res := testutil.InitTestWorld(t, testutil.WithUI()).Resources.UIResources
	rect := image.Rect(0, 0, 400, 400)
	modal := buildPanelUI(res, rect, DetailContent{Name: "Biscuit", Desc: "A hard biscuit that keeps well as a preserved food.", Rows: storySpecRows()}, page)
	screen := ebiten.NewImage(400, 400)
	modal.Draw(uicore.NewEbitenCanvas(screen))
	vrt.AssertFrameGolden(t, name, screen)
}

func TestGolden_Story_DetailFirstPage(t *testing.T) {
	t.Parallel()
	drawDetailStory(t, "TestGolden_Story_DetailFirstPage", 0)
}

func TestGolden_Story_DetailLastPage(t *testing.T) {
	t.Parallel()
	drawDetailStory(t, "TestGolden_Story_DetailLastPage", 1)
}
