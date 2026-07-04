package styled

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

// BackgroundStyle は背景描画のスタイル設定を表す
type BackgroundStyle struct {
	BorderColor     color.RGBA // 枠線の色
	BackgroundColor color.RGBA // 背景の色
	BorderWidth     float32    // 枠線の太さ
	HighlightColor  color.RGBA // 上辺ハイライト線の色。ゼロ値なら描画しない
	ShadowColor     color.RGBA // 下辺シャドウ線の色。ゼロ値なら描画しない
}

// DrawFramedBackground は枠線付きの背景を描画する。
// HighlightColor/ShadowColorが設定されている場合、上辺/下辺に1pxの線を追加して立体感を出す
func DrawFramedBackground(screen *ebiten.Image, x, y, width, height int, style BackgroundStyle) {
	// 枠線を描画
	if style.BorderWidth > 0 {
		vector.StrokeRect(screen,
			float32(x),
			float32(y),
			float32(width),
			float32(height),
			style.BorderWidth,
			style.BorderColor,
			false)
	}

	// 内側の背景を描画（枠線を避けるため少し小さくする）
	borderOffset := max(int(style.BorderWidth), 1)

	vector.FillRect(screen,
		float32(x+borderOffset),
		float32(y+borderOffset),
		float32(width-borderOffset*2),
		float32(height-borderOffset*2),
		style.BackgroundColor,
		false)

	// 上辺ハイライト線（1px）
	if style.HighlightColor.A > 0 {
		vector.FillRect(screen,
			float32(x+borderOffset),
			float32(y+borderOffset),
			float32(width-borderOffset*2),
			1,
			style.HighlightColor,
			false)
	}

	// 下辺シャドウ線（1px）
	if style.ShadowColor.A > 0 {
		vector.FillRect(screen,
			float32(x+borderOffset),
			float32(y+height-borderOffset-1),
			float32(width-borderOffset*2),
			1,
			style.ShadowColor,
			false)
	}
}

// PanelStyle はゲーム内パネルの標準スタイルを返す
func PanelStyle() BackgroundStyle {
	return BackgroundStyle{
		BackgroundColor: theme.PanelBackground,
		BorderColor:     theme.PanelHighlight,
		BorderWidth:     1,
		HighlightColor:  theme.PanelHighlight,
		ShadowColor:     theme.PanelShadow,
	}
}
