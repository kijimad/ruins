package gamelog

import "image/color"

// rgb はRGB値からcolor.RGBAを作成する
func rgb(v uint64) color.RGBA {
	return color.RGBA{
		R: uint8((v >> 16) & 0xFF),
		G: uint8((v >> 8) & 0xFF),
		B: uint8(v & 0xFF),
		A: 0xFF,
	}
}

// ゲームログで使用するパレット色
var (
	ColorWhite   = rgb(0xFFFFFF)
	ColorBlack   = rgb(0x000000)
	ColorRed     = rgb(0xFF0000)
	ColorGreen   = rgb(0x00FF00)
	ColorBlue    = rgb(0x0000FF)
	ColorYellow  = rgb(0xFFFF00)
	ColorCyan    = rgb(0x00FFFF)
	ColorMagenta = rgb(0xFF00FF)
	ColorOrange  = rgb(0xFFA500)
	ColorPurple  = rgb(0x800080)
)
