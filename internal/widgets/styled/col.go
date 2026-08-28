package styled

// ColMode は列の幅の決め方。
type ColMode int

// 列の幅の決め方。
const (
	// ColGrow は余り幅を吸収して伸びる列。名前列に使う
	ColGrow ColMode = iota
	// ColFit は内容の実測幅に合わせる列。数値やタグに使う
	ColFit
	// ColIcon はアイコン用の正方の固定列
	ColIcon
)

// Col は一覧の1列の定義。幅は宣言せず、伸縮 Grow・内容実測 Fit・正方 Icon の様式だけを持つ。
// 実測は行データを知る一覧側が行う。
type Col struct {
	Mode  ColMode
	Align TextAlign
}

// Name は名前やラベル用の列を返す。余り幅を吸収して伸び、左寄せで描く。
func Name() Col { return Col{Mode: ColGrow, Align: AlignLeft} }

// Fit はタグや状態表示用の列を返す。内容の実測幅に合わせ、左寄せで描く。
func Fit() Col { return Col{Mode: ColFit, Align: AlignLeft} }

// Num は数値・重量・価格用の列を返す。内容の実測幅に合わせ、右寄せで桁を揃える。
func Num() Col { return Col{Mode: ColFit, Align: AlignRight} }

// Icon はアイテムのアイコン用の正方の列を返す。
func Icon() Col { return Col{Mode: ColIcon, Align: AlignLeft} }

// Cols は列の並びをそのまま返す。呼び出し側の宣言を読みやすくする可変長版。
func Cols(cs ...Col) []Col { return cs }

// Aligns は列の並びから揃えだけを取り出す。
func Aligns(cols []Col) []TextAlign {
	as := make([]TextAlign, len(cols))
	for i, c := range cols {
		as[i] = c.Align
	}
	return as
}
