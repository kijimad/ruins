package entityspec

import (
	"image/color"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// specLabelColor はラベルと見出しの色。
var specLabelColor color.Color = theme.SpecLabel

// SpecRowWidgets は SpecRow の並びを uicore の行ウィジェット列にする。
// ラベルは左寄せ、値は右寄せ。見出し行はラベルを行の全幅で描き、色付き行は値をその色で描く。
// ラベル列の幅は全行の実測から決め、値列が余り幅を吸って行の右端に揃う。幅の数値は持たない。
// モーダルはこの列に name や desc の行を足して1枚のパネルに組む。
func SpecRowWidgets(rows []SpecRow, face text.Face) []uicore.Widget {
	labels := make([]int, 0, len(rows))
	for _, r := range rows {
		if !r.Header {
			labels = append(labels, uicore.MeasureTextWidth(r.Label, face))
		}
	}
	cols := []int{uicore.FitWidth(labels, theme.Space3, 0), 0}

	items := make([]uicore.Widget, 0, len(rows))
	for _, r := range rows {
		if r.Header {
			items = append(items, uicore.Row([]int{0}, uicore.NewText(r.Label, face, specLabelColor)))
			continue
		}
		valColor := specLabelColor
		if r.Color != nil {
			valColor = r.Color
		}
		value := uicore.NewText(r.Value, face, valColor)
		value.Align = uicore.AlignRight
		label := uicore.NewText(r.Label, face, specLabelColor)
		if r.Indent {
			// 見出しの下の子行を1段下げる。先頭に空の桁を差してラベルだけ右へずらし、値は右端揃えのまま
			items = append(items, uicore.Row([]int{theme.Space4, cols[0], cols[1]}, uicore.NewGroup(), label, value))
			continue
		}
		items = append(items, uicore.Row(cols, label, value))
	}
	return items
}

// BuildSpecPanel は SpecRow の並びから uicore の保持型ツリーを組む。
// 必要なのは本文フェイスだけなのでそれだけを受け取る。行高は渡されたフェイスの行送りにする。
// 行は文字を収める箱なので、字面を切らない高さはフェイスから導ける。固定値では持たない。
// パッケージグローバルの可変状態に触れないので、複数の UI を並行に組んでも競合しない。
func BuildSpecPanel(rows []SpecRow, face text.Face) uicore.Widget {
	return uicore.VBox(uicore.LineHeight(face), SpecRowWidgets(rows, face)...)
}
