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

func TestGraphic_画像があれば描く(t *testing.T) {
	t.Parallel()
	img := ebiten.NewImage(4, 4)
	g := uicore.NewGraphic(img)
	g.Layout(image.Rect(0, 0, 20, 20))

	cv := &recordCanvas{}
	g.Draw(cv)

	require.Len(t, cv.images, 1, "画像があれば描画呼び出しが1回起きる")
	assert.Nil(t, g.Children(), "子は持たない")
}

func TestGraphic_画像が無ければ何も描かない(t *testing.T) {
	t.Parallel()
	g := uicore.NewGraphic(nil)
	g.Layout(image.Rect(0, 0, 20, 20))

	cv := &recordCanvas{}
	g.Draw(cv)

	assert.Empty(t, cv.images, "nil画像は描画呼び出しをしない")
}

func TestNineSlice_画像があれば描く(t *testing.T) {
	t.Parallel()
	img := ebiten.NewImage(9, 9)
	n := uicore.NewNineSlice(img, [3]int{3, 3, 3}, [3]int{3, 3, 3})
	n.Layout(image.Rect(0, 0, 30, 30))

	cv := &recordCanvas{}
	n.Draw(cv)

	require.Len(t, cv.images, 1, "画像があれば描画呼び出しが1回起きる")
	assert.Nil(t, n.Children(), "子は持たない")
}

func TestNineSlice_画像が無ければ何も描かない(t *testing.T) {
	t.Parallel()
	n := uicore.NewNineSlice(nil, [3]int{}, [3]int{})
	n.Layout(image.Rect(0, 0, 30, 30))

	cv := &recordCanvas{}
	n.Draw(cv)

	assert.Empty(t, cv.images, "nil画像は描画呼び出しをしない")
}

func TestGroup_子を配置し直さず順に描く(t *testing.T) {
	t.Parallel()
	a := uicore.NewText("a", nil, color.White)
	b := uicore.NewText("b", nil, color.White)
	g := uicore.NewGroup(a, b)

	require.Equal(t, []uicore.Widget{a, b}, g.Children(), "渡した子をそのまま返す")

	g.Layout(image.Rect(0, 0, 10, 10))
	cv := &recordCanvas{}
	g.Draw(cv)

	assert.Equal(t, []string{"a", "b"}, cv.texts, "子を渡した順に描く")
}
