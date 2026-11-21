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

// Draw は Widget を満たす。比率ぶんの塗りを枠の内側へ敷き、枠を重ねる。比率は 0..1 に収める。
// 比率が正なら最低1pxを塗り、残りわずかでも尽きたようには見せない
func (g *Gauge) Draw(cv Canvas) {
	// 塗りは枠の線の内側に置く。枠の線と重なると、最後の1pxが枠に隠れる
	inner := g.rect.Inset(1)
	ratio := max(0, min(1, g.Ratio))
	fillW := int(float64(inner.Dx()) * ratio)
	if ratio > 0 {
		fillW = max(fillW, 1)
	}
	if fillW > 0 && g.Fill != nil {
		dst := image.Rect(inner.Min.X, inner.Min.Y, inner.Min.X+fillW, inner.Max.Y)
		cv.DrawImageTintedRect(dst, g.Fill, color.NRGBA(g.FillColor))
	}
	cv.StrokeRect(g.rect, 1, g.Border, RectOptions{Radius: g.Radius})
}

// Children は Widget を満たす。子は持たない。
func (g *Gauge) Children() []Widget { return nil }
