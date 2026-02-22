package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// OutlinedText は枠線付きテキストを描画する
// 8方向にオフセットして枠線を描画した後、本体のテキストを描画する
func OutlinedText(screen *ebiten.Image, str string, face text.Face, x, y float64, textColor, outlineColor color.Color) {
	offsets := []struct{ dx, dy float64 }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	op := &text.DrawOptions{}
	for _, offset := range offsets {
		op.GeoM.Reset()
		op.GeoM.Translate(x+offset.dx, y+offset.dy)
		op.ColorScale.Reset()
		op.ColorScale.ScaleWithColor(outlineColor)
		text.Draw(screen, str, face, op)
	}

	op.GeoM.Reset()
	op.GeoM.Translate(x, y)
	op.ColorScale.Reset()
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, str, face, op)
}
