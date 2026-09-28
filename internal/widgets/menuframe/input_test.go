package menuframe_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

func TestInputBox_ChildrenとBoundsでWidgetの契約を満たす(t *testing.T) {
	t.Parallel()

	res := resources.UIResources{InputBG: &resources.NineSliceTex{Image: ebiten.NewImage(4, 4)}}
	body := uicore.NewText("Ash", nil, theme.TextPrimary)

	box := menuframe.InputBox(res, body)

	children := box.Children()
	require.Len(t, children, 2, "枠と中身の2つを子に持つ")
	assert.Equal(t, body, children[1], "2番目の子は渡したbody")

	rect := image.Rect(0, 0, 320, 44)
	box.Layout(rect)

	bounder, ok := box.(interface{ Bounds() image.Rectangle })
	require.True(t, ok, "InputBoxはBoundsを実装する")
	assert.Equal(t, rect, bounder.Bounds(), "Layoutでもらった矩形をそのまま保持する")
}
