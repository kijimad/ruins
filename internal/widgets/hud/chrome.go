package hud

import (
	"image"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
)

// Chrome は世界の上に重ねるパネルの意匠。メニューと同じテクスチャを使い、
// 画面をまたいで枠の見た目を1つにする。
//
// 枠を図形で組むと、同じ「パネル」という概念にメニューと HUD で2つの描き方ができてしまう。
// 意匠をテクスチャに一本化すれば、差し替えも1箇所で済む。
type Chrome struct {
	panel *resources.NineSliceTex
}

// NewChrome はパネルのテクスチャを与えて意匠を作る。
func NewChrome(res resources.UIResources) Chrome {
	return Chrome{panel: res.PanelBG}
}

// Panel は矩形へパネルの枠と背景を敷く。
func (c Chrome) Panel(cv ui.Canvas, r image.Rectangle) {
	if c.panel == nil {
		return
	}
	cv.DrawNineSlice(r, c.panel.Image, c.panel.BX, c.panel.BY)
}
