package entityspec

import (
	"image/color"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/ui"
)

// specPanelCols は spec パネルの列幅。ラベル列と値列。
var specPanelCols = []int{70, 80}

// specPanelRowH は spec パネルの行高。
const specPanelRowH = 18

// specLabelColor はラベルと見出しの色。
var specLabelColor color.Color = color.White

// SpecPanelRowH は spec パネル1行の高さ。モーダル側が name や desc を同じ行高で並べるのに使う。
const SpecPanelRowH = specPanelRowH

// SpecRowWidgets は SpecRow の並びを internal/ui の行ウィジェット列にする。
// ラベルは左寄せ、値は右寄せ。見出し行はラベルのみを2列ぶんの幅で描き、色付き行は値をその色で描く。
// モーダルはこの列に name や desc の行を足して1枚のパネルに組む。
func SpecRowWidgets(rows []SpecRow, face text.Face) []ui.Widget {
	items := make([]ui.Widget, 0, len(rows))
	for _, r := range rows {
		if r.Header {
			span := []int{specPanelCols[0] + specPanelCols[1]}
			items = append(items, ui.Row(span, ui.NewText(r.Label, face, specLabelColor)))
			continue
		}
		valColor := specLabelColor
		if r.Color != nil {
			valColor = r.Color
		}
		value := ui.NewText(r.Value, face, valColor)
		value.Align = ui.AlignRight
		items = append(items, ui.Row(specPanelCols, ui.NewText(r.Label, face, specLabelColor), value))
	}
	return items
}

// BuildSpecPanel は SpecRow の並びから internal/ui の保持型ツリーを組む。
// 必要なのは本文フェイスだけなのでそれだけを受け取る。
// パッケージグローバルの可変状態に触れないので、複数の UI を並行に組んでも競合しない。
func BuildSpecPanel(rows []SpecRow, face text.Face) *ui.Container {
	return ui.VBox(specPanelRowH, SpecRowWidgets(rows, face)...)
}
