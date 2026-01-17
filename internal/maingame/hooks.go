package maingame

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	w "github.com/kijimaD/ruins/internal/world"
)

// ブラー画像と黒背景のキャッシュ
var (
	cachedBlurImage       *ebiten.Image
	cachedBlackBackground *ebiten.Image
	cachedStateCount      int
)

// afterDrawHook は各stateのDraw完了後に呼ばれるフック関数
func afterDrawHook(stateIndex, stateCount int, _ w.World, screen *ebiten.Image) error {
	if cachedStateCount != stateCount {
		// キャッシュをクリア
		cachedBlurImage = nil
		cachedStateCount = stateCount
	}

	// stateが複数あるとき初回のstate描画後に1回だけブラー効果を適用する
	if stateCount > 1 && stateIndex == 0 {
		bounds := screen.Bounds()

		// キャッシュがない場合のみブラー処理を実行
		if cachedBlurImage == nil {
			// 現在の画面をコピー
			src := ebiten.NewImage(bounds.Dx(), bounds.Dy())
			src.DrawImage(screen, nil)

			// ブラー効果を2パス（水平→垂直）で適用
			// 水平ブラー
			const blurRadius = 4
			tmp := ebiten.NewImage(bounds.Dx(), bounds.Dy())
			for x := -blurRadius; x <= blurRadius; x++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(x), 0)
				op.ColorScale.ScaleAlpha(1.0 / float32(blurRadius*2+1))
				op.Blend = ebiten.BlendSourceOver
				tmp.DrawImage(src, op)
			}

			// 垂直ブラー
			cachedBlurImage = ebiten.NewImage(bounds.Dx(), bounds.Dy())
			for y := -blurRadius; y <= blurRadius; y++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(0, float64(y))
				op.ColorScale.ScaleAlpha(1.0 / float32(blurRadius*2+1))
				op.Blend = ebiten.BlendSourceOver
				cachedBlurImage.DrawImage(tmp, op)
			}
		}

		// 黒い背景画像をキャッシュから取得または生成
		if cachedBlackBackground == nil {
			cachedBlackBackground = ebiten.NewImage(bounds.Dx(), bounds.Dy())
			width := float32(bounds.Dx())
			height := float32(bounds.Dy())
			vector.FillRect(cachedBlackBackground, 0, 0, width, height, color.RGBA{0, 0, 0, 255}, false)
		}

		screen.Clear()

		// 黒背景を描画
		screen.DrawImage(cachedBlackBackground, nil)

		// その上にブラー画像を描画
		screen.DrawImage(cachedBlurImage, nil)
	}

	return nil
}
