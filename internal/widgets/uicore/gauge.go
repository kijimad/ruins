package uicore

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Gauge は1本のゲージ。与えられた矩形へ枠を引き、比率ぶんの塗りをグラデーションの
// テクスチャで色掛けする。上が明るく下が暗い光沢になる。
type Gauge struct {
	base
	Fill      *ebiten.Image
	Ratio     float64
	FillColor color.RGBA
	Border    color.RGBA
	Radius    int
}

// Layout は Widget を満たす。
func (g *Gauge) Layout(r image.Rectangle) { g.rect = r }

// Draw は Widget を満たす。比率ぶんの塗りを枠内へ敷き、枠を重ねる。
func (g *Gauge) Draw(cv Canvas) {
	fillW := int(float64(g.rect.Dx()) * g.Ratio)
	if fillW > 0 && g.Fill != nil {
		// 塗りは上下1pxだけ内へ寄せ、枠の線と角の丸みに塗りが食み出さないようにする
		dst := image.Rect(g.rect.Min.X, g.rect.Min.Y+1, g.rect.Min.X+fillW, g.rect.Max.Y-1)
		cv.DrawImageTintedRect(dst, g.Fill, color.NRGBA(g.FillColor))
	}
	cv.StrokeRect(g.rect, 1, g.Border, RectOptions{Radius: g.Radius})
}

// Children は Widget を満たす。子は持たない。
func (g *Gauge) Children() []Widget { return nil }
