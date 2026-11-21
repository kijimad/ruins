package menuframe

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// gaugeCanvas はアイコンとゲージの描いた矩形を記録する Canvas
type gaugeCanvas struct {
	icons   []image.Rectangle
	tinted  []image.Rectangle
	strokes []image.Rectangle
}

func (c *gaugeCanvas) FillRect(image.Rectangle, color.Color, uicore.RectOptions) {}
func (c *gaugeCanvas) FillTriangle(_, _, _ [2]float32, _ color.Color)            {}
func (c *gaugeCanvas) StrokeRect(r image.Rectangle, _ int, _ color.Color, _ uicore.RectOptions) {
	c.strokes = append(c.strokes, r)
}
func (c *gaugeCanvas) DrawText(image.Point, string, text.Face, color.Color) {}
func (c *gaugeCanvas) DrawImage(image.Point, *ebiten.Image)                 {}
func (c *gaugeCanvas) DrawImageRect(dst image.Rectangle, _ *ebiten.Image) {
	c.icons = append(c.icons, dst)
}
func (c *gaugeCanvas) DrawNineSlice(image.Rectangle, *ebiten.Image, [3]int, [3]int) {}
func (c *gaugeCanvas) DrawImageTintedRect(dst image.Rectangle, _ *ebiten.Image, _ color.Color) {
	c.tinted = append(c.tinted, dst)
}

func TestIconCell(t *testing.T) {
	t.Parallel()
	icon := ebiten.NewImage(20, 20)
	fill := ebiten.NewImage(1, 4)
	tests := []struct {
		name        string
		cell        styled.Cell
		rect        image.Rectangle
		wantTinted  []image.Rectangle
		wantStrokes []image.Rectangle
	}{
		{
			name: "ゲージが無ければアイコンだけを描く",
			cell: styled.IconCell(icon),
			rect: image.Rect(0, 0, 20, 21),
		},
		{
			name:        "ゲージはアイコンの下端にアイコンの幅で重ねる",
			cell:        styled.Cell{Icon: icon, Gauge: &styled.CellGauge{Ratio: 0.5}},
			rect:        image.Rect(0, 0, 20, 21),
			wantTinted:  []image.Rectangle{image.Rect(1, 17, 10, 19)},
			wantStrokes: []image.Rectangle{image.Rect(0, 16, 20, 20)},
		},
		{
			name:        "アイコンが縮むとゲージも縮んだアイコンに合わせる",
			cell:        styled.Cell{Icon: icon, Gauge: &styled.CellGauge{Ratio: 0.5}},
			rect:        image.Rect(0, 0, 40, 10),
			wantTinted:  []image.Rectangle{image.Rect(1, 7, 5, 9)},
			wantStrokes: []image.Rectangle{image.Rect(0, 6, 10, 10)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := iconCell(tt.cell, fill)
			w.Layout(tt.rect)
			cv := &gaugeCanvas{}
			w.Draw(cv)
			require.Equal(t, []image.Rectangle{tt.rect}, cv.icons)
			assert.Equal(t, tt.wantTinted, cv.tinted)
			assert.Equal(t, tt.wantStrokes, cv.strokes)
		})
	}
}
