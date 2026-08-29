package menuframe

import (
	"image"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// inputBox は1行入力欄の箱。テクスチャ枠を全面に敷き、中身は左右に余白を取って同じ矩形へ載せる。
// 中身の縦位置は Text の VCenter で枠の中央へ合わせる。
type inputBox struct {
	rect  image.Rectangle
	frame uicore.Widget
	body  uicore.Widget
}

// InputBox は1行入力欄の箱を組む。名前入力のように、テクスチャ枠の上へ入力中の文字列を
// 載せる画面で使う。枠と余白の意匠は部品が持ち、置き場所は呼び出し側が Layout で決める。
func InputBox(res resources.UIResources, body uicore.Widget) uicore.Widget {
	return &inputBox{
		frame: uicore.NewNineSlice(res.InputBG.Image, res.InputBG.BX, res.InputBG.BY),
		body:  body,
	}
}

// Layout は Widget を実装する。枠を全面に、中身を左右余白の内側に置く。
func (b *inputBox) Layout(r image.Rectangle) {
	b.rect = r
	b.frame.Layout(r)
	b.body.Layout(image.Rect(r.Min.X+theme.Space4, r.Min.Y, r.Max.X-theme.Space4, r.Max.Y))
}

// Draw は Widget を実装する。枠を敷いてから中身を描く。
func (b *inputBox) Draw(cv uicore.Canvas) {
	b.frame.Draw(cv)
	b.body.Draw(cv)
}

// Children は Widget を実装する。
func (b *inputBox) Children() []uicore.Widget { return []uicore.Widget{b.frame, b.body} }

// Bounds は Widget を実装する。
func (b *inputBox) Bounds() image.Rectangle { return b.rect }
