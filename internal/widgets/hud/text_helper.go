package hud

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// OutlinedText は枠線付きテキストを描く。8方向へずらして枠を描いてから本体を重ねる。
// 世界の上に文字を置くので、背景の明暗によらず読めるようにする
func OutlinedText(cv uicore.Canvas, str string, face text.Face, pos image.Point, textColor, outlineColor color.Color) {
	offsets := []image.Point{
		{X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1},
		{X: -1, Y: 0}, {X: 1, Y: 0},
		{X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1},
	}
	for _, off := range offsets {
		cv.DrawText(pos.Add(off), str, face, outlineColor)
	}
	cv.DrawText(pos, str, face, textColor)
}

// drawOutlinedText は共通の縁色で枠線付きテキストを描く。HUD の文字はすべてこれを通す
func drawOutlinedText(cv uicore.Canvas, textStr string, face text.Face, pos image.Point, textColor color.Color) {
	OutlinedText(cv, textStr, face, pos, textColor, theme.HUDTextOutline)
}
