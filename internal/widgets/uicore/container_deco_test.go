package uicore_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainer_SetPadding_内側に余白を作る(t *testing.T) {
	t.Parallel()
	child := uicore.VBox(0)
	child.SetStyle(uicore.BoxStyle{Fill: color.White})
	parent := uicore.VBox(80, child)
	parent.SetPadding(10)

	parent.Layout(image.Rect(0, 0, 100, 100))
	cv := &recordCanvas{}
	parent.Draw(cv)

	require.Len(t, cv.fills, 1)
	assert.Equal(t, image.Pt(10, 10), cv.fills[0].Min, "paddingぶん内側から子が配置される")
}

func TestContainer_SetBackgroundNineSlice_背景画像を描く(t *testing.T) {
	t.Parallel()
	img := ebiten.NewImage(9, 9)
	c := uicore.VBox(0)
	c.SetBackgroundNineSlice(img, [3]int{3, 3, 3}, [3]int{3, 3, 3})

	c.Layout(image.Rect(0, 0, 30, 30))
	cv := &recordCanvas{}
	c.Draw(cv)

	assert.Len(t, cv.images, 1, "9スライス背景が描かれる")
}

func TestContainer_SetBottomLine_下端に線を敷く(t *testing.T) {
	t.Parallel()
	img := ebiten.NewImage(10, 2)
	c := uicore.VBox(0)
	c.SetBottomLine(img, color.White)

	c.Layout(image.Rect(0, 0, 50, 20))
	cv := &recordCanvas{}
	c.Draw(cv)

	require.Len(t, cv.images, 1)
	assert.Equal(t, image.Pt(0, 18), cv.images[0], "線画像の高さぶん上げた位置、下端に敷かれる")
}

func TestContainer_Row_0幅の列は余り幅を均等に吸収する(t *testing.T) {
	t.Parallel()
	a := uicore.VBox(0)
	a.SetStyle(uicore.BoxStyle{Fill: color.White})
	b := uicore.VBox(0)
	b.SetStyle(uicore.BoxStyle{Fill: color.White})
	row := uicore.Row([]int{0, 0}, a, b)

	row.Layout(image.Rect(0, 0, 100, 10))
	cv := &recordCanvas{}
	row.Draw(cv)

	require.Len(t, cv.fills, 2)
	assert.Equal(t, 50, cv.fills[0].Dx(), "1列目は余り幅を均等に吸収する")
	assert.Equal(t, 50, cv.fills[1].Dx(), "2列目も同様に吸収する")
}
