package uicore_test

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordDrawable は Widget を実装しない Drawable。Placeable が包むことを確かめる。
type recordDrawable struct {
	drawCount int
}

func (d *recordDrawable) Draw(_ uicore.Canvas) { d.drawCount++ }

func TestPlaceable_Widget実装はそのまま通す(t *testing.T) {
	t.Parallel()
	w := uicore.NewText("a", nil, nil)
	out := uicore.Placeable([]uicore.Drawable{w})

	require.Len(t, out, 1)
	assert.Same(t, w, out[0], "Widgetを実装していればラップされずそのまま返る")
}

func TestPlaceable_非Widgetは描画専用として包む(t *testing.T) {
	t.Parallel()
	d := &recordDrawable{}
	out := uicore.Placeable([]uicore.Drawable{d})

	require.Len(t, out, 1)
	w := out[0]
	assert.Nil(t, w.Children(), "包んだ葉は子を持たない")

	w.Layout(image.Rect(0, 0, 10, 10))
	w.Draw(nil)
	assert.Equal(t, 1, d.drawCount, "DrawはPlaceableで包んだ中身へ委譲される")
}
