package styled

import (
	"image"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// GradientLine は両端がグラデーションで透明になる水平線ウィジェット
type GradientLine struct {
	widget *widget.Widget
	color  color.RGBA
	height int
	src    *ebiten.Image
}

// NewGradientLine はグラデーション線ウィジェットを作成する。
// src はグラデーションパターンの画像アセットで、描画時に幅に合わせてスケーリングされる
func NewGradientLine(src *ebiten.Image, clr color.RGBA, height int) *GradientLine {
	w := widget.NewWidget(
		widget.WidgetOpts.LayoutData(widget.RowLayoutData{
			Stretch: true,
		}),
		widget.WidgetOpts.MinSize(0, height),
	)
	return &GradientLine{
		widget: w,
		color:  clr,
		height: height,
		src:    src,
	}
}

// GetWidget はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) GetWidget() *widget.Widget {
	return g.widget
}

// SetLocation はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) SetLocation(rect image.Rectangle) {
	g.widget.Rect = rect
}

// PreferredSize はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) PreferredSize() (int, int) {
	return 0, g.height
}

// Render はグラデーション線を描画する
func (g *GradientLine) Render(screen *ebiten.Image) {
	if !g.widget.IsVisible() || g.src == nil {
		return
	}
	g.widget.Render(screen)

	rect := g.widget.Rect
	w := rect.Dx()
	if w <= 0 {
		return
	}

	op := &ebiten.DrawImageOptions{}
	srcWidth := float64(g.src.Bounds().Dx())
	op.GeoM.Scale(float64(w)/srcWidth, float64(g.height))
	op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	op.ColorScale.ScaleWithColor(color.NRGBA(g.color))
	screen.DrawImage(g.src, op)
}

// Update はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) Update(updateObj *widget.UpdateObject) {
	g.widget.Update(updateObj)
}

// Validate はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) Validate() {}

// IsValidated はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) IsValidated() bool { return true }
