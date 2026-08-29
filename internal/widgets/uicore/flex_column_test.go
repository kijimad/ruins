package uicore_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlexColumn_Growの行が余り高さを吸収する(t *testing.T) {
	t.Parallel()
	header := uicore.VBox(0)
	header.SetStyle(uicore.BoxStyle{Fill: color.White})
	body := uicore.VBox(0)
	body.SetStyle(uicore.BoxStyle{Fill: color.White})
	footer := uicore.VBox(0)
	footer.SetStyle(uicore.BoxStyle{Fill: color.White})

	uicore.FlexColumn(image.Rect(0, 0, 50, 100), []uicore.FlexItem{
		{W: header, Height: 20},
		{W: body, Grow: true},
		{W: footer, Height: 10},
	})

	cv := &recordCanvas{}
	header.Draw(cv)
	body.Draw(cv)
	footer.Draw(cv)

	require.Len(t, cv.fills, 3)
	assert.Equal(t, 20, cv.fills[0].Dy(), "先頭行は固定高で確定する")
	assert.Equal(t, 70, cv.fills[1].Dy(), "Growの行が余り高さを吸収する")
	assert.Equal(t, 10, cv.fills[2].Dy(), "末尾行は固定高で確定する")
	assert.Equal(t, 90, cv.fills[2].Min.Y, "末尾行は下端に押し付けられる")
}
