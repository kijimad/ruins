package styled

import "github.com/kijimaD/ruins/internal/widgets/theme"

// Col は一覧の1列の定義。幅と揃えを持つ。生のピクセル幅の代わりに意味的な構成関数で宣言し、
// 「名前列」「数値列」「アイコン列」のように役割で列を組めるようにする。
type Col struct {
	Width int
	Align TextAlign
}

// Name は名前やラベル用の左寄せ列を幅 w で返す。
func Name(w int) Col { return Col{Width: w, Align: AlignLeft} }

// Num は数値・重量・価格用の右寄せ列を幅 w で返す。桁を右端で揃える。
func Num(w int) Col { return Col{Width: w, Align: AlignRight} }

// Icon はアイテムのアイコン用の正方の左寄せ列を返す。幅は theme のアイコン列幅に従う。
func Icon() Col { return Col{Width: theme.MenuIconW, Align: AlignLeft} }

// Cols は列の並びをそのまま返す。呼び出し側の宣言を読みやすくする可変長版。
func Cols(cs ...Col) []Col { return cs }

// Widths は列の並びから幅だけを取り出す。
func Widths(cols []Col) []int {
	ws := make([]int, len(cols))
	for i, c := range cols {
		ws[i] = c.Width
	}
	return ws
}

// Aligns は列の並びから揃えだけを取り出す。
func Aligns(cols []Col) []TextAlign {
	as := make([]TextAlign, len(cols))
	for i, c := range cols {
		as[i] = c.Align
	}
	return as
}
