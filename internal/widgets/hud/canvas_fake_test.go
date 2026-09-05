package hud

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// fakeCanvas は uicore.Canvas の記録用実装。ebiten の描画コンテキスト無しで
// どの描画命令が何回・どの引数で呼ばれたかだけを検証する。
type fakeCanvas struct {
	texts       []textCall
	fillRects   []image.Rectangle
	strokeRects []image.Rectangle
	nineSlices  int
	tintedRects []image.Rectangle
}

// textCall は DrawText 呼び出し1回ぶんの記録
type textCall struct {
	pos   image.Point
	str   string
	color color.Color
}

func (c *fakeCanvas) FillRect(r image.Rectangle, _ color.Color) {
	c.fillRects = append(c.fillRects, r)
}

func (c *fakeCanvas) StrokeRect(r image.Rectangle, _ int, _ color.Color) {
	c.strokeRects = append(c.strokeRects, r)
}

func (c *fakeCanvas) DrawText(pos image.Point, s string, _ text.Face, col color.Color) {
	c.texts = append(c.texts, textCall{pos: pos, str: s, color: col})
}

func (c *fakeCanvas) DrawImage(_ image.Point, _ *ebiten.Image) {}

func (c *fakeCanvas) DrawImageRect(_ image.Rectangle, _ *ebiten.Image) {}

func (c *fakeCanvas) DrawNineSlice(_ image.Rectangle, _ *ebiten.Image, _, _ [3]int) {
	c.nineSlices++
}

func (c *fakeCanvas) DrawImageTintedRect(dst image.Rectangle, _ *ebiten.Image, _ color.Color) {
	c.tintedRects = append(c.tintedRects, dst)
}
