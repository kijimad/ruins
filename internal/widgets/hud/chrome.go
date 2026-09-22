package hud

import (
	"image"

	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// Chrome は世界の上に重ねるパネルの意匠。角丸の塗りと枠で描き、画面をまたいで見た目を1つにする。
// 意匠の差し替えも1箇所で済ませる。状態は持たないので Chrome{} で作る。
type Chrome struct{}

// Panel は矩形へパネルの角丸背景と枠を敷く。
func (c Chrome) Panel(cv uicore.Canvas, r image.Rectangle) {
	cv.FillRect(r, theme.PanelBackground, uicore.RectOptions{Radius: theme.CornerRadius})
	cv.StrokeRect(r, 1, theme.PanelHighlight, uicore.RectOptions{Radius: theme.CornerRadius})
}
