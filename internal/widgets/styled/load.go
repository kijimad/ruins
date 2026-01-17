package styled

import (
	e_image "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// LoadButtonImage はボタンイメージを読み込む
// TODO: いい感じにしたい
func LoadButtonImage() *widget.ButtonImage {
	idle := e_image.NewNineSliceColor(consts.ButtonIdleColor)
	hover := e_image.NewNineSliceColor(consts.ButtonHoverColor)
	pressed := e_image.NewNineSliceColor(consts.ButtonPressedColor)
	pressedHover := e_image.NewNineSliceColor(consts.ButtonPressedColor)
	disabled := e_image.NewNineSliceColor(consts.ButtonPressedColor)

	return &widget.ButtonImage{
		Idle:         idle,
		Hover:        hover,
		Pressed:      pressed,
		PressedHover: pressedHover,
		Disabled:     disabled,
	}
}

// LoadFont はフォントを読み込む
// TODO: いい感じにしたい
func LoadFont(world w.World) *text.Face {
	face := (*world.Resources.Faces)["kappa"]

	return &face
}
