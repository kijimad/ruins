package balance

import (
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// mdAlignSpec は列整列を1文字で表す。L=左、R=右、C=中央。数値は R、区分やラベルは L、
// 判定など記号は C にする。列数と同じ長さの文字列で writeMDTable へ渡す。
func mdAlignSpec(spec string) []tw.Align {
	aligns := make([]tw.Align, 0, len(spec))
	for _, c := range spec {
		switch c {
		case 'R':
			aligns = append(aligns, tw.AlignRight)
		case 'C':
			aligns = append(aligns, tw.AlignCenter)
		default:
			aligns = append(aligns, tw.AlignLeft)
		}
	}
	return aligns
}

// writeMDTable は列見出しと行から markdown 表を1つ書き出し、末尾に空行を1つ足す。align は列ごとの
// 整列を表す文字列で、数値の右寄せや判定の中央寄せをそのまま GFM の区切りへ反映する。列数・整列・
// 区切りを tablewriter へ集約し、手書きの縦棒とセパレータがずれるのを防ぐ。見出しの自動整形は日本語を
// 壊すため切り、自動折返しも切って1セル1行にする。
func writeMDTable(b *strings.Builder, header []string, align string, rows [][]string) {
	aligns := mdAlignSpec(align)
	t := tablewriter.NewTable(b,
		tablewriter.WithRenderer(renderer.NewMarkdown()),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
		tablewriter.WithHeaderAlignmentConfig(tw.CellAlignment{PerColumn: aligns}),
		tablewriter.WithRowAlignmentConfig(tw.CellAlignment{PerColumn: aligns}),
	)
	t.Header(toAnySlice(header)...)
	for _, r := range rows {
		_ = t.Append(toAnySlice(r)...)
	}
	_ = t.Render()
	b.WriteString("\n")
}

// toAnySlice は文字列スライスを tablewriter の可変長引数へ渡すため []any へ変換する。
func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
