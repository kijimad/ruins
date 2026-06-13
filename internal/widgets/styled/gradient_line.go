package styled

import (
	"image"
	"image/color"
	"math"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// GradientLine は両端がグラデーションで透明になる水平線ウィジェット
type GradientLine struct {
	widget *widget.Widget
	color  color.RGBA
	height int
	cache  *ebiten.Image
}

// NewGradientLine はグラデーション線ウィジェットを作成する
func NewGradientLine(clr color.RGBA, height int) *GradientLine {
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
	if !g.widget.IsVisible() {
		return
	}
	g.widget.Render(screen)

	rect := g.widget.Rect
	w := rect.Dx()
	if w <= 0 {
		return
	}

	// キャッシュがない、または幅が変わった場合に再生成する
	// 色をピクセルに直接焼き込むことでプリマルチプライドアルファを正しく保つ
	if g.cache == nil || g.cache.Bounds().Dx() != w {
		g.cache = ebiten.NewImage(w, g.height)
		pixels := make([]byte, w*g.height*4)
		cr := float64(g.color.R) / 255.0
		cg := float64(g.color.G) / 255.0
		cb := float64(g.color.B) / 255.0
		ca := float64(g.color.A) / 255.0
		fadePixels := w / 4 // 両端25%ずつグラデーション
		for px := 0; px < w; px++ {
			t := 1.0
			if fadePixels > 0 {
				if px < fadePixels {
					t = float64(px) / float64(fadePixels)
				} else if px >= w-fadePixels {
					t = float64(w-1-px) / float64(fadePixels)
				}
			}
			a := t * ca
			for py := 0; py < g.height; py++ {
				i := (py*w + px) * 4
				pixels[i] = byte(math.Round(a * cr * 255))
				pixels[i+1] = byte(math.Round(a * cg * 255))
				pixels[i+2] = byte(math.Round(a * cb * 255))
				pixels[i+3] = byte(math.Round(a * 255))
			}
		}
		g.cache.WritePixels(pixels)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	screen.DrawImage(g.cache, op)
}

// Update はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) Update(updateObj *widget.UpdateObject) {
	g.widget.Update(updateObj)
}

// Validate はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) Validate() {}

// IsValidated はwidget.PreferredSizeLocateableWidget インターフェースを満たす
func (g *GradientLine) IsValidated() bool { return true }
