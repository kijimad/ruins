package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/kijimaD/ruins/internal/consts"
)

// TileFrame はタイルを囲う枠を描く。corners は投影済みの四隅で、並びは北西・北東・南東・南西。
//
// 透視投影ではタイルが台形になるため、正方形の画像を貼ると形が合わない。
// 投影済みの4点をそのまま線で結ぶことで、実際に描かれたタイルの輪郭と一致する。
func TileFrame(screen *ebiten.Image, corners [4]consts.Coord[consts.ScreenPixel], width float32, clr color.Color) {
	for i := range corners {
		from, to := corners[i], corners[(i+1)%len(corners)]
		vector.StrokeLine(screen,
			float32(from.X), float32(from.Y),
			float32(to.X), float32(to.Y),
			width, clr, true)
	}
}

// ScaleAlpha は色のアルファに係数を掛ける。カーソルの点滅など、色はそのままで濃さだけ変える用途に使う。
func ScaleAlpha(clr color.RGBA, alpha float64) color.RGBA {
	clr.A = uint8(float64(clr.A) * min(max(alpha, 0), 1))
	return clr
}
