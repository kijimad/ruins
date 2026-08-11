package menuframe

import (
	"image"

	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// logTopY はゲームログ領域の上端Yを返す。モーダルやパネルをログに被らせないための基準にする
func logTopY(screenHeight int) int {
	cfg := hud.DefaultMessageAreaConfig
	logHeight := cfg.LogAreaMargin*2 + cfg.MaxLogLines*cfg.LineHeight + cfg.YPadding*2
	return screenHeight - logHeight - theme.Space3
}

// CenterWindowRect はゲームワールドの画面サイズから、ログ上端より上に収まる中央のウィンドウ矩形を返す。
// overlay の小窓を置く共通の位置で、画面ごとに位置合わせをしなくてよいようにする
func CenterWindowRect(world w.World) image.Rectangle {
	windowWidth, windowHeight := 400, 400

	screenWidth := world.Resources.ScreenDimensions.Width
	screenHeight := world.Resources.ScreenDimensions.Height

	// 横は画面中央。縦はゲームログの上端より上の領域に収めて、ログと重ならないようにする
	x := screenWidth/2 - windowWidth/2
	logTop := logTopY(screenHeight)
	y := max((logTop-windowHeight)/2, theme.Space3)

	return image.Rect(x, y, x+windowWidth, y+windowHeight)
}
