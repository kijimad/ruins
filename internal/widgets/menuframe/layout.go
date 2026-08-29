package menuframe

import (
	"image"

	"github.com/kijimaD/ruins/internal/widgets/hud"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// logTopY はゲームログ領域の上端Yを返す。モーダルやパネルをログに被らせないための基準にする
func logTopY(screenHeight int) int {
	return screenHeight - hud.DefaultMessageAreaConfig.Height() - theme.Space3
}

// ModalRect は大きめモーダル、タブ画面など、の矩形を返す。画面をひと回り小さくし、ログ領域の
// 手前で止める。上端は他のウィンドウと同じ MenuWindowTop で揃え、下はログ予約ぶんを空ける。
func ModalRect(world w.World) image.Rectangle {
	sw := world.Resources.ScreenDimensions.Width
	sh := world.Resources.ScreenDimensions.Height
	logReserve := sh - logTopY(sh) + theme.Space3
	return image.Rect(theme.MenuModalMarginX, theme.MenuWindowTop, sw-theme.MenuModalMarginX, sh-logReserve)
}

// WindowRect は小さめウィンドウの標準矩形を返す。詳細モーダルやパネルを置く共通の位置で、
// 画面ごとに位置合わせをしなくてよいようにする。横は画面中央、上端は他のウィンドウと同じ
// MenuWindowTop で揃える
func WindowRect(world w.World) image.Rectangle {
	windowWidth, windowHeight := 400, 400
	screenWidth := world.Resources.ScreenDimensions.Width
	x := screenWidth/2 - windowWidth/2
	y := theme.MenuWindowTop
	return image.Rect(x, y, x+windowWidth, y+windowHeight)
}
