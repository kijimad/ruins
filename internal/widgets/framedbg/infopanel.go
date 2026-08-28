package framedbg

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// 画面右上へ置く情報パネルの寸法
const (
	infoPanelWidth  = 300 // パネルの幅
	infoPanelMargin = 10  // 画面端との余白
	infoPanelPad    = 10  // パネル内側の余白
	// LineH は即時描画のパネルに置く本文1行の高さ。ログ領域も同じ値を使い、
	// 世界の上に重ねるパネルの行送りを揃える
	LineH = 20
)

// InfoPanel は画面の右上へ置く情報パネル。枠付き背景を敷き、行を上から順に書き足す。
// 見回しや射撃のように、ゲーム世界の上へ一時的に情報を出す画面で使う。
// 保持型のツリーを組まず即時に描くので、世界の描画と同じ流れに乗せられる。
type InfoPanel struct {
	screen *ebiten.Image
	face   text.Face
	rect   image.Rectangle
	y      int // 次に書き込む行の上端
}

// NewInfoPanel は高さ height のパネルを画面右上へ敷き、書き込み位置を先頭に置く。
func NewInfoPanel(screen *ebiten.Image, face text.Face, height int) *InfoPanel {
	x := screen.Bounds().Dx() - infoPanelWidth - infoPanelMargin
	rect := image.Rect(x, infoPanelMargin, x+infoPanelWidth, infoPanelMargin+height)
	Draw(screen, rect.Min.X, rect.Min.Y, rect.Dx(), rect.Dy(), PanelStyle())
	return &InfoPanel{
		screen: screen,
		face:   face,
		rect:   rect,
		y:      rect.Min.Y + infoPanelPad,
	}
}

// Line は1行書き、書き込み位置を1行ぶん送る。
func (p *InfoPanel) Line(s string) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(p.rect.Min.X+infoPanelPad), float64(p.y))
	text.Draw(p.screen, s, p.face, op)
	p.y += LineH
}

// Gap は書き込み位置を px ぶん送る。段落の切れ目に使う。
func (p *InfoPanel) Gap(px int) { p.y += px }

// SeekBottom は書き込み位置をパネル下端から fromBottom だけ上へ移す。
// 操作説明のように、内容の量によらず下端に置きたい行に使う。
func (p *InfoPanel) SeekBottom(fromBottom int) { p.y = p.rect.Max.Y - fromBottom }
