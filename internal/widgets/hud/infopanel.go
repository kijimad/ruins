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

// NewInfoPanel は高さ height のパネルを画面右上へ敷き、書き込み位置を先頭に置く。
func NewInfoPanel(cv uicore.Canvas, chrome Chrome, face text.Face, screenWidth, height int) *InfoPanel {
	x := screenWidth - infoPanelWidth - infoPanelMargin
	rect := image.Rect(x, infoPanelMargin, x+infoPanelWidth, infoPanelMargin+height)
	chrome.Panel(cv, rect)
	return &InfoPanel{
		cv:   cv,
		face: face,
		rect: rect,
		y:    rect.Min.Y + infoPanelPad,
	}
}

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
