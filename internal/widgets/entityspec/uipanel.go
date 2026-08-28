package entityspec

import (
	"image/color"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

// specPanelRowH は spec パネルの行高。
const specPanelRowH = 18

// specLabelColor はラベルと見出しの色。
var specLabelColor color.Color = theme.SpecLabel

// SpecPanelRowH は spec パネル1行の高さ。モーダル側が name や desc を同じ行高で並べるのに使う。
const SpecPanelRowH = specPanelRowH

// SpecRowWidgets は SpecRow の並びを internal/ui の行ウィジェット列にする。
// ラベルは左寄せ、値は右寄せ。見出し行はラベルを行の全幅で描き、色付き行は値をその色で描く。
// ラベル列の幅は全行の実測から決め、値列が余り幅を吸って行の右端に揃う。幅の数値は持たない。
// モーダルはこの列に name や desc の行を足して1枚のパネルに組む。
func SpecRowWidgets(rows []SpecRow, face text.Face) []ui.Widget {
	labels := make([]int, 0, len(rows))
	for _, r := range rows {
		if !r.Header {
			labels = append(labels, ui.MeasureTextWidth(r.Label, face))
		}
	}
	cols := []int{ui.FitWidth(labels, theme.Space3, 0), 0}

	items := make([]ui.Widget, 0, len(rows))
	for _, r := range rows {
		if r.Header {
			items = append(items, ui.Row([]int{0}, ui.NewText(r.Label, face, specLabelColor)))
			continue
		}
		valColor := specLabelColor
		if r.Color != nil {
			valColor = r.Color
		}
		value := ui.NewText(r.Value, face, valColor)
		value.Align = ui.AlignRight
		items = append(items, ui.Row(cols, ui.NewText(r.Label, face, specLabelColor), value))
	}
	return items
}

// BuildSpecPanel は SpecRow の並びから internal/ui の保持型ツリーを組む。
// 必要なのは本文フェイスだけなのでそれだけを受け取る。
// パッケージグローバルの可変状態に触れないので、複数の UI を並行に組んでも競合しない。
func BuildSpecPanel(rows []SpecRow, face text.Face) ui.Widget {
	return ui.VBox(specPanelRowH, SpecRowWidgets(rows, face)...)
}
