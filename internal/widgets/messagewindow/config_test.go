package messagewindow

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig_主要フィールドの初期値を検証(t *testing.T) {
	t.Parallel()

	cfg := defaultWindowConfig()

	assert.Equal(t, windowSize{Width: MinWidth, Height: MinHeight}, cfg.Size)
	assert.True(t, cfg.Center)

	assert.Equal(t, theme.WindowBackground, cfg.windowStyle.BackgroundColor)
	assert.Equal(t, theme.WindowBorder, cfg.windowStyle.BorderColor)
	assert.Equal(t, 2, cfg.windowStyle.BorderWidth)
	assert.Equal(t, windowPadding{Top: 20, Bottom: 20, Left: 20, Right: 20}, cfg.windowStyle.windowPadding)

	assert.Equal(t, theme.TextPrimary, cfg.textStyle.Color)
	assert.Equal(t, 24, cfg.textStyle.LineHeight)

	assert.True(t, cfg.actionStyle.ShowCloseButton)
	assert.Equal(t, "Close [Enter/Escape]", cfg.actionStyle.CloseButtonText)
	assert.Equal(t, theme.WindowActionBg, cfg.actionStyle.ActionAreaColor)
	assert.Equal(t, theme.WindowActionText, cfg.actionStyle.ActionTextColor)

	assert.ElementsMatch(t, []ebiten.Key{ebiten.KeyEnter, ebiten.KeyEscape, ebiten.KeySpace}, cfg.SkippableKeys)
	assert.False(t, cfg.CloseOnClick)
	assert.True(t, cfg.ShowBackground)
}
