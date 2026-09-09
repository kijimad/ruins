package hud

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrencyDisplay_Draw_無効なら何も描かない(t *testing.T) {
	t.Parallel()
	display := NewCurrencyDisplay(nil)
	display.SetEnabled(false)
	cv := &fakeCanvas{}

	display.Draw(cv, CurrencyData{
		Currency:         consts.Currency(100),
		ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
		Config:           DefaultMessageAreaConfig,
	})

	assert.Empty(t, cv.texts)
}

func TestCurrencyDisplay_Draw_有効なら通貨をログ領域の上に描く(t *testing.T) {
	t.Parallel()
	display := NewCurrencyDisplay(nil)
	cv := &fakeCanvas{}

	data := CurrencyData{
		Currency:         consts.Currency(1234),
		ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
		Config:           DefaultMessageAreaConfig,
	}
	display.Draw(cv, data)

	require.NotEmpty(t, cv.texts)
	last := cv.texts[len(cv.texts)-1]
	assert.Equal(t, data.Currency.String(), last.str)

	logAreaY := data.ScreenDimensions.Height - data.Config.Height()
	assert.Less(t, last.pos.Y, logAreaY, "ログ領域より上に描く")
}
