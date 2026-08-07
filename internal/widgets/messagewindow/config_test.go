package messagewindow

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig_主要フィールドの初期値を検証(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	assert.Equal(t, WindowSize{Width: MinWidth, Height: MinHeight}, cfg.Size)
	assert.True(t, cfg.Center)

	assert.Equal(t, theme.WindowBackground, cfg.WindowStyle.BackgroundColor)
	assert.Equal(t, theme.WindowBorder, cfg.WindowStyle.BorderColor)
	assert.Equal(t, 2, cfg.WindowStyle.BorderWidth)
	assert.Equal(t, Padding{Top: 20, Bottom: 20, Left: 20, Right: 20}, cfg.WindowStyle.Padding)

	assert.Equal(t, theme.TextPrimary, cfg.TextStyle.Color)
	assert.Equal(t, 24, cfg.TextStyle.LineHeight)

	assert.True(t, cfg.ActionStyle.ShowCloseButton)
	assert.Equal(t, "閉じる [Enter/Escape]", cfg.ActionStyle.CloseButtonText)
	assert.Equal(t, theme.WindowActionBg, cfg.ActionStyle.ActionAreaColor)
	assert.Equal(t, theme.WindowActionText, cfg.ActionStyle.ActionTextColor)

	assert.ElementsMatch(t, []ebiten.Key{ebiten.KeyEnter, ebiten.KeyEscape, ebiten.KeySpace}, cfg.SkippableKeys)
	assert.False(t, cfg.CloseOnClick)
	assert.True(t, cfg.ShowBackground)
}
