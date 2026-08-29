package menuframe

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
)

func TestLogTopY(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		screenHeight int
		want         int
	}{
		{"720pxの画面ではログ領域140pxとSpace3を差し引く", 720, 572},
		{"1000pxの画面でも同じ差し引き幅になる", 1000, 852},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, logTopY(tt.screenHeight))
		})
	}
}

func TestWindowRect_横は画面中央で上端はMenuWindowTopに揃える(t *testing.T) {
	t.Parallel()
	world := w.World{Resources: &resources.Resources{
		ScreenDimensions: resources.ScreenDimensions{Width: 960, Height: 720},
	}}

	got := WindowRect(world)

	assert.Equal(t, image.Rect(280, theme.MenuWindowTop, 680, theme.MenuWindowTop+400), got)
}
