package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// drawOutlinedText は枠線付きテキストを描画する
// textColorを指定することで任意の色でテキストを描画できる
func drawOutlinedText(screen *ebiten.Image, textStr string, face text.Face, x, y float64, textColor color.Color) {
	// 黒い枠線を描画（8方向に少しずらして描画）
	offsets := []struct{ dx, dy float64 }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for _, offset := range offsets {
		op := &text.DrawOptions{}
		op.GeoM.Translate(x+offset.dx, y+offset.dy)
		op.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 255}) // 黒
		text.Draw(screen, textStr, face, op)
	}

	// テキスト本体を描画
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, textStr, face, op)
}
