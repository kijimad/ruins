package screenui

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/resources"
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
			assert.Equal(t, tt.want, LogTopY(tt.screenHeight))
		})
	}
}

func TestCenterWindowRect_画面中央にログ領域を避けてウィンドウ矩形を配置する(t *testing.T) {
	t.Parallel()
	world := w.World{Resources: &resources.Resources{
		ScreenDimensions: resources.ScreenDimensions{Width: 960, Height: 720},
	}}

	got := CenterWindowRect(world)

	assert.Equal(t, image.Rect(280, 86, 680, 486), got)
}

func TestCenterWindowRect_ログ領域が広く上端が窮屈なときはSpace3にクランプする(t *testing.T) {
	t.Parallel()
	world := w.World{Resources: &resources.Resources{
		ScreenDimensions: resources.ScreenDimensions{Width: 800, Height: 100},
	}}

	got := CenterWindowRect(world)

	assert.Equal(t, image.Rect(200, 8, 600, 408), got)
}
