package hud

import (
	"image"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// Chrome は世界の上に重ねるパネルの意匠。角丸の塗りと枠で描き、画面をまたいで見た目を1つにする。
// 意匠の差し替えも1箇所で済ませる。
type Chrome struct{}

// NewChrome はパネルの意匠を作る。意匠は角丸の単色描画で UI リソースは参照しないが、
// 呼び出し側の取り回しを揃えるため引数の形は保つ。
func NewChrome(_ resources.UIResources) Chrome {
	return Chrome{}
}

// Panel は矩形へパネルの角丸背景と枠を敷く。
func (c Chrome) Panel(cv uicore.Canvas, r image.Rectangle) {
	cv.FillRect(r, theme.PanelBackground, theme.CornerRadius)
	cv.StrokeRect(r, 1, theme.PanelHighlight, theme.CornerRadius)
}
