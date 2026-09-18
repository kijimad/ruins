package balance

import (
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// 列整列の別名。数値は右、区分やラベルは左、判定など記号は中央にする。tablewriter の定数への別名。
const (
	alignL = tw.AlignLeft
	alignR = tw.AlignRight
	alignC = tw.AlignCenter
)

// writeMDTable は tablewriter の markdown レンダラで表を1つ書き、末尾に空行を足す。整列は列ごとに渡す。
// 見出しの自動整形と自動折返しは日本語を壊すため切る。表描画自体は tablewriter に委ねる。
func writeMDTable(b *strings.Builder, header []string, aligns []tw.Align, rows [][]string) {
	t := tablewriter.NewTable(b,
		tablewriter.WithRenderer(renderer.NewMarkdown()),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
		tablewriter.WithHeaderAlignmentConfig(tw.CellAlignment{PerColumn: aligns}),
		tablewriter.WithRowAlignmentConfig(tw.CellAlignment{PerColumn: aligns}),
	)
	t.Header(header)
	_ = t.Bulk(rows)
	_ = t.Render()
	b.WriteString("\n")
}
