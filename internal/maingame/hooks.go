package maingame

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

var (
	cachedBlurImage   *ebiten.Image
	cachedDarkOverlay *ebiten.Image
	cachedStateCount  int
)

// applyBlurOverlay は画面にブラー効果を適用する
// stateCountが変わるとキャッシュを再生成する
// applyBlurがfalseの場合はキャッシュ更新のみ行う
func applyBlurOverlay(screen *ebiten.Image, stateCount int, applyBlur bool) {
	if stateCount != cachedStateCount {
		cachedBlurImage = nil
		cachedStateCount = stateCount
	}

	if !applyBlur {
		return
	}

	bounds := screen.Bounds()

	if cachedBlurImage == nil {
		w, h := bounds.Dx(), bounds.Dy()

		src := ebiten.NewImage(w, h)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(-bounds.Min.X), float64(-bounds.Min.Y))
		src.DrawImage(screen, op)

		const blurRadius = 4
		kernel := blurRadius*2 + 1

		tmp := ebiten.NewImage(w, h)
		for x := -blurRadius; x <= blurRadius; x++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), 0)
			op.ColorScale.Scale(1.0/float32(kernel), 1.0/float32(kernel), 1.0/float32(kernel), 1.0/float32(kernel))
			op.Blend = ebiten.BlendLighter
			tmp.DrawImage(src, op)
		}

		cachedBlurImage = ebiten.NewImage(w, h)
		for y := -blurRadius; y <= blurRadius; y++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, float64(y))
			op.ColorScale.Scale(1.0/float32(kernel), 1.0/float32(kernel), 1.0/float32(kernel), 1.0/float32(kernel))
			op.Blend = ebiten.BlendLighter
			cachedBlurImage.DrawImage(tmp, op)
		}
	}

	if cachedDarkOverlay == nil {
		cachedDarkOverlay = ebiten.NewImage(bounds.Dx(), bounds.Dy())
		cachedDarkOverlay.Fill(color.RGBA{0, 0, 0, 48})
	}

	drawOp := &ebiten.DrawImageOptions{}
	drawOp.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	screen.DrawImage(cachedBlurImage, drawOp)
	screen.DrawImage(cachedDarkOverlay, drawOp)
}
