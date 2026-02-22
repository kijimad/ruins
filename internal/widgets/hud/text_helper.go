package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/widgets/render"
)

// drawOutlinedText は枠線付きテキストを描画する
// textColorを指定することで任意の色でテキストを描画できる
func drawOutlinedText(screen *ebiten.Image, textStr string, face text.Face, x, y float64, textColor color.Color) {
	render.OutlinedText(screen, textStr, face, x, y, textColor, color.RGBA{0, 0, 0, 255})
}
