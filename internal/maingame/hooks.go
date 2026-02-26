package maingame

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
)

// ブラー画像と暗転オーバーレイのキャッシュ
var (
	cachedBlurImage   *ebiten.Image
	cachedDarkOverlay *ebiten.Image
	cachedBaseState   es.State[w.World]
)

// applyBlurOverlay は画面にブラー効果を適用する
// baseStateが変わるとキャッシュを再生成する
func applyBlurOverlay(screen *ebiten.Image, baseState es.State[w.World]) {
	// 下層stateが変わったらキャッシュをクリア
	if cachedBaseState != baseState {
		cachedBlurImage = nil
		cachedBaseState = baseState
	}

	bounds := screen.Bounds()

	// キャッシュがない場合のみブラー画像を生成する
	if cachedBlurImage == nil {
		w, h := bounds.Dx(), bounds.Dy()

		// 現在の画面をコピーする
		src := ebiten.NewImage(w, h)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(-bounds.Min.X), float64(-bounds.Min.Y))
		src.DrawImage(screen, op)

		// ブラー効果を2パス（水平→垂直）で適用する
		const blurRadius = 4
		kernel := blurRadius*2 + 1

		// 水平ブラー
		tmp := ebiten.NewImage(bounds.Dx(), bounds.Dy())
		for x := -blurRadius; x <= blurRadius; x++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x), 0)
			op.ColorScale.Scale(1.0/float32(kernel), 1.0/float32(kernel), 1.0/float32(kernel), 1.0/float32(kernel))
			op.Blend = ebiten.BlendLighter
			tmp.DrawImage(src, op)
		}

		// 垂直ブラー
		cachedBlurImage = ebiten.NewImage(bounds.Dx(), bounds.Dy())
		for y := -blurRadius; y <= blurRadius; y++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, float64(y))
			op.ColorScale.Scale(1.0/float32(kernel), 1.0/float32(kernel), 1.0/float32(kernel), 1.0/float32(kernel))
			op.Blend = ebiten.BlendLighter
			cachedBlurImage.DrawImage(tmp, op)
		}
	}

	// 黒い半透明オーバーレイをキャッシュから取得または生成する
	if cachedDarkOverlay == nil {
		cachedDarkOverlay = ebiten.NewImage(bounds.Dx(), bounds.Dy())
		cachedDarkOverlay.Fill(color.RGBA{0, 0, 0, 48})
	}

	drawOp := &ebiten.DrawImageOptions{}
	drawOp.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	screen.DrawImage(cachedBlurImage, drawOp)
	screen.DrawImage(cachedDarkOverlay, drawOp)
}
