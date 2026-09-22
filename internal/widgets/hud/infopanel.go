package hud

import (
	"image"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// 画面右上へ置く情報パネルの寸法
const (
	infoPanelWidth  = 300 // パネルの幅
	infoPanelMargin = 10  // 画面端との余白
	infoPanelPad    = 10  // パネル内側の余白
	// LineH は世界の上に重ねるパネルに置く本文1行の高さ。ログ領域も同じ値を使い、行送りを揃える
	LineH = 20
)

// InfoPanel は画面の右上へ置く情報パネル。パネルの意匠を敷き、行を上から順に書き足す。
// 見回しや射撃のように、ゲーム世界の上へ一時的に情報を出す画面で使う。
type InfoPanel struct {
	cv   uicore.Canvas
	face text.Face
	rect image.Rectangle
	y    int // 次に書き込む行の上端
}

// NewInfoPanel は高さ height のパネルを画面右上へ敷き、書き込み位置を先頭に置く。パネルの右上配置は
// FlexColumn/Row へ委ね、画面幅からの逆算をしない。行はパネル矩形を基準に上から書き足す。
func NewInfoPanel(cv uicore.Canvas, chrome Chrome, face text.Face, screenWidth, height int) *InfoPanel {
	// uicore は隅への固定サイズ配置ヘルパを持たないので、rectHolder で flex の配置結果の矩形だけを取り出す。
	// 先頭0幅列で右寄せし、1行だけの FlexColumn で上端へ置いて右上のパネル矩形を得る
	holder := &rectHolder{}
	row := uicore.Row([]int{0, infoPanelWidth}, uicore.NewGroup(), holder)
	uicore.FlexColumn(
		image.Rect(0, infoPanelMargin, screenWidth-infoPanelMargin, infoPanelMargin+height),
		[]uicore.FlexItem{{W: row, Height: height}},
	)
	rect := holder.rect
	chrome.Panel(cv, rect)
	return &InfoPanel{
		cv:   cv,
		face: face,
		rect: rect,
		y:    rect.Min.Y + infoPanelPad,
	}
}

// rectHolder は FlexColumn/Row から確定矩形を受け取るだけの Widget。自身は何も描かず、隅へ固定サイズの
// パネルを置くときに配置結果の矩形を取り出すのに使う。
type rectHolder struct{ rect image.Rectangle }

// Layout は uicore.Widget を満たす。
func (h *rectHolder) Layout(r image.Rectangle) { h.rect = r }

// Draw は uicore.Widget を満たす。何も描かない。
func (h *rectHolder) Draw(uicore.Canvas) {}

// Children は uicore.Widget を満たす。子は持たない。
func (h *rectHolder) Children() []uicore.Widget { return nil }

// Line は1行書き、書き込み位置を1行ぶん送る。
func (p *InfoPanel) Line(s string) {
	p.cv.DrawText(image.Pt(p.rect.Min.X+infoPanelPad, p.y), s, p.face, theme.TextPrimary)
	p.y += LineH
}

// Gap は書き込み位置を px ぶん送る。段落の切れ目に使う。
func (p *InfoPanel) Gap(px int) { p.y += px }

// SeekBottom は書き込み位置をパネル下端から fromBottom だけ上へ移す。
// 操作説明のように、内容の量によらず下端に置きたい行に使う。
func (p *InfoPanel) SeekBottom(fromBottom int) { p.y = p.rect.Max.Y - fromBottom }
