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
	// DrawImageRect は img を dst に収まるよう縦横比を保って縮小し、拡大はせず、左寄せ・縦中央で描く。
	// 一覧のアイコンやキーキャップを行の高さへ合わせ、左に揃えて縦中央に置くのに使う。
	DrawImageRect(dst image.Rectangle, img *ebiten.Image)
	// DrawNineSlice は img を9スライスで dst いっぱいに引き伸ばして描く。四隅は原寸、辺は片方向、
	// 中央は両方向へ伸びる。bx・by はソースの左中右・上中下の各スライス幅。枠付き背景に使う。
	DrawNineSlice(dst image.Rectangle, img *ebiten.Image, bx, by [3]int)
	// DrawImageTintedRect は img を dst いっぱいに引き伸ばし、tint で色を掛けて描く。
	// 横グラデーションのテクスチャを行幅へ伸ばし、色を付けて一覧の区切り線にするのに使う。
	DrawImageTintedRect(dst image.Rectangle, img *ebiten.Image, tint color.Color)
}

// BoxStyle は矩形の塗りと枠。塗りと枠はそれぞれ色が nil なら描かない。
type BoxStyle struct {
	Fill        color.Color
	Border      color.Color
	BorderWidth int
}
