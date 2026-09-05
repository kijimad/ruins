package hud

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoPanel(t *testing.T) {
	t.Parallel()

	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	chrome := NewChrome(res)

	cv := &fakeCanvas{}
	const screenWidth = 800
	const height = 200
	panel := NewInfoPanel(cv, chrome, res.Text.BodyFace, screenWidth, height)

	assert.Equal(t, 1, cv.nineSlices, "生成時にパネルの意匠を1回敷く")

	panel.Line("1行目")
	panel.Line("2行目")
	require.Len(t, cv.texts, 2)
	assert.Equal(t, "1行目", cv.texts[0].str)
	assert.Equal(t, theme.TextPrimary, cv.texts[0].color)
	assert.Equal(t, LineH, cv.texts[1].pos.Y-cv.texts[0].pos.Y, "Lineは呼ぶたびに1行ぶん書き込み位置を下げる")

	panel.Gap(30)
	panel.Line("3行目")
	require.Len(t, cv.texts, 3)
	assert.Equal(t, cv.texts[1].pos.Y+30+LineH, cv.texts[2].pos.Y, "Gapは指定pxだけ書き込み位置を送る")

	wantRect := image.Rect(screenWidth-infoPanelWidth-infoPanelMargin, infoPanelMargin, screenWidth-infoPanelMargin, infoPanelMargin+height)
	panel.SeekBottom(15)
	panel.Line("末尾行")
	require.Len(t, cv.texts, 4)
	assert.Equal(t, wantRect.Max.Y-15, cv.texts[3].pos.Y, "SeekBottomはパネル下端からfromBottomだけ上へ書き込み位置を移す")
	assert.Equal(t, wantRect.Min.X+infoPanelPad, cv.texts[3].pos.X)
}
