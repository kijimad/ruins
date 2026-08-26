package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Canvas は描画先を抽象化する seam。UI ツリーはこの seam 越しに描く。
// テストは記録用の実装を渡し、ebiten の描画コンテキスト無しでレイアウトとテキストを検証できる。
// 本番は ebiten 実装 EbitenCanvas を渡す。
type Canvas interface {
	// FillRect は矩形を塗る。
	FillRect(r image.Rectangle, c color.Color)
	// StrokeRect は矩形の枠を width の太さで描く。
	StrokeRect(r image.Rectangle, width int, c color.Color)
	// DrawText は pos を左上として1行を描く。
	DrawText(pos image.Point, s string, face text.Face, c color.Color)
	// DrawImage は pos を左上として画像を描く。
	DrawImage(pos image.Point, img *ebiten.Image)
}

// BoxStyle は矩形の塗りと枠。塗りと枠はそれぞれ色が nil なら描かない。
type BoxStyle struct {
	Fill        color.Color
	Border      color.Color
	BorderWidth int
}
