package menuframe

import (
	"image"
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
)

// TestPanelInner_内側の矩形を返す は、外枠の矩形からメニュー余白ぶん縮めた内側矩形を返す
// 純関数を固定する。
func TestPanelInner_内側の矩形を返す(t *testing.T) {
	t.Parallel()

	got := PanelInner(image.Rect(0, 0, 100, 100))
	assert.Equal(t, image.Rect(theme.MenuPad, theme.MenuPad, 100-theme.MenuPad, 100-theme.MenuPad), got)
}

// TestSplitScreen_タイトル説明ヒントと左右を含む は、左右2枠の画面を組んだツリーに
// タイトル・説明・ヒントと左右の中身のラベルが並ぶことを固定する。
func TestSplitScreen_タイトル説明ヒントと左右を含む(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	res := world.Resources.UIResources
	left := uicore.NewText("左", res.Text.SmallFace, color.White)
	right := uicore.NewText("右", res.Text.SmallFace, color.White)

	tree := SplitScreen(world, res, "タイトル", left, right, "説明", "ヒント")

	labels := uicore.CollectLabels(tree)
	assert.Subset(t, labels, []string{"タイトル", "説明", "ヒント", "左", "右"})
}

// TestFormScreen_タイトル本文ヒントを含む は、フォーム画面を組んだツリーに要素のラベルが
// 並ぶことを固定する。エラー文の有無で行が増減する分岐も検証する。
func TestFormScreen_タイトル本文ヒントを含む(t *testing.T) {
	t.Parallel()

	t.Run("エラー文ありは全要素を含む", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t, testutil.WithUI())
		res := world.Resources.UIResources
		body := uicore.NewText("本文", res.Text.SmallFace, color.White)

		tree := FormScreen(world, res, "フォーム", body, "エラー", "ヒント")

		labels := uicore.CollectLabels(tree)
		assert.Subset(t, labels, []string{"フォーム", "本文", "エラー", "ヒント"})
	})

	t.Run("エラー文なしはエラー行を含まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t, testutil.WithUI())
		res := world.Resources.UIResources
		body := uicore.NewText("本文", res.Text.SmallFace, color.White)

		tree := FormScreen(world, res, "フォーム", body, "", "ヒント")

		labels := uicore.CollectLabels(tree)
		assert.Subset(t, labels, []string{"フォーム", "本文", "ヒント"})
		assert.NotContains(t, labels, "エラー")
	})
}
