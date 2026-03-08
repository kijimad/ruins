package hud

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

// drawOutlinedText は枠線付きテキストを描画する
// textColorを指定することで任意の色でテキストを描画できる
func drawOutlinedText(screen *ebiten.Image, textStr string, face text.Face, x, y float64, textColor color.Color) {
	OutlinedText(screen, textStr, face, x, y, textColor, color.RGBA{0, 0, 0, 255})
}
