package hud

import (
	"image"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/internal/uicore"
)

// Chrome は世界の上に重ねるパネルの意匠。メニューと同じテクスチャを使い、
// 画面をまたいで枠の見た目を1つにする。図形の手組みで別の描き方が増えるのを防ぎ、
// 意匠の差し替えも1箇所で済ませる。
type Chrome struct {
	panel *resources.NineSliceTex
}

// NewChrome はパネルのテクスチャを与えて意匠を作る。
func NewChrome(res resources.UIResources) Chrome {
	return Chrome{panel: res.PanelBG}
}

// Panel は矩形へパネルの枠と背景を敷く。
func (c Chrome) Panel(cv uicore.Canvas, r image.Rectangle) {
	if c.panel == nil {
		return
	}
	cv.DrawNineSlice(r, c.panel.Image, c.panel.BX, c.panel.BY)
}
