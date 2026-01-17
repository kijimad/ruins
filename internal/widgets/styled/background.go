package styled

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// BackgroundStyle は背景描画のスタイル設定を表す
type BackgroundStyle struct {
	BorderColor     color.RGBA // 枠線の色
	BackgroundColor color.RGBA // 背景の色
	BorderWidth     float32    // 枠線の太さ
}

// DefaultMessageBackgroundStyle はデフォルトのメッセージ背景スタイルを返す
func DefaultMessageBackgroundStyle() BackgroundStyle {
	return BackgroundStyle{
		BorderColor:     color.RGBA{255, 255, 255, 255}, // 白色の枠線
		BackgroundColor: color.RGBA{0, 0, 0, 200},       // 半透明の黒背景
		BorderWidth:     2,                              // 線の太さ
	}
}

// DrawFramedBackground は枠線付きの背景を描画する
func DrawFramedBackground(screen *ebiten.Image, x, y, width, height int, style BackgroundStyle) {
	// 枠線を描画
	vector.StrokeRect(screen,
		float32(x),
		float32(y),
		float32(width),
		float32(height),
		style.BorderWidth,
		style.BorderColor,
		false)

	// 内側の背景を描画（枠線を避けるため少し小さくする）
	borderOffset := int(style.BorderWidth)
	if borderOffset < 1 {
		borderOffset = 1
	}

	vector.FillRect(screen,
		float32(x+borderOffset),
		float32(y+borderOffset),
		float32(width-borderOffset*2),
		float32(height-borderOffset*2),
		style.BackgroundColor,
		false)
}
