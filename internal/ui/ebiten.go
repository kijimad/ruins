package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// EbitenCanvas は Canvas を ebiten の描画で満たす本番実装。
type EbitenCanvas struct {
	screen *ebiten.Image
}

var _ Canvas = (*EbitenCanvas)(nil)

// NewEbitenCanvas は描画先スクリーンを与えて Canvas を作る。
func NewEbitenCanvas(screen *ebiten.Image) *EbitenCanvas {
	return &EbitenCanvas{screen: screen}
}

// FillRect は EbitenCanvas を実装する。
func (e *EbitenCanvas) FillRect(r image.Rectangle, c color.Color) {
	vector.FillRect(e.screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
}

// StrokeRect は EbitenCanvas を実装する。
func (e *EbitenCanvas) StrokeRect(r image.Rectangle, width int, c color.Color) {
	vector.StrokeRect(e.screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), float32(width), c, false)
}

// DrawText は EbitenCanvas を実装する。pos を左上として1行を描く。
func (e *EbitenCanvas) DrawText(pos image.Point, s string, face text.Face, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(pos.X), float64(pos.Y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(e.screen, s, face, op)
}

// DrawImage は EbitenCanvas を実装する。pos を左上として画像を描く。
func (e *EbitenCanvas) DrawImage(pos image.Point, img *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(pos.X), float64(pos.Y))
	e.screen.DrawImage(img, op)
}
