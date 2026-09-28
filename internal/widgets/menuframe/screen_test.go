package menuframe_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

func TestPanelInner_内側余白ぶん矩形を縮める(t *testing.T) {
	t.Parallel()

	rect := image.Rect(0, 0, 100, 60)

	inner := menuframe.PanelInner(rect)

	want := image.Rect(theme.MenuPad, theme.MenuPad, 100-theme.MenuPad, 60-theme.MenuPad)
	assert.Equal(t, want, inner)
}

func TestImagePanel_画像をパネル内側へ配置する(t *testing.T) {
	t.Parallel()

	res := resources.UIResources{}
	img := ebiten.NewImage(10, 10)
	rect := image.Rect(0, 0, 100, 60)

	panel := menuframe.ImagePanel(res, rect, img)

	assert.Len(t, panel.Children(), 2, "背景とアイコンの2つを子に持つ")
}
