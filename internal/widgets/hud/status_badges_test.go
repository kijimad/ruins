package hud

import (
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/stretchr/testify/assert"
)

func TestBadgeStyle_背景色だけ上書きしパネルスタイルを引き継ぐ(t *testing.T) {
	t.Parallel()

	fill := color.RGBA{R: 200, G: 30, B: 30, A: 255}

	got := badgeStyle(fill)

	want := styled.PanelStyle()
	want.BackgroundColor = fill
	assert.Equal(t, want, got)
}
