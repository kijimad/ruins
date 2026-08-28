package ui

import (
	"math"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// MeasureText は face で描いたときの s の送り幅と高さを画素で返す。
//
// フォントの送り幅はほぼ整数だが、わずかに端数を持つ。切り捨てると境界をまたぐたびに
// 1px 揺れ、列の幅や中央寄せの位置がずれる。四捨五入で境界から遠ざけ、丸め方を
// この1箇所に固定する。寸法を内容から決めたい箇所はすべてここを通す。
//
// フェイスが nil なら測れないので 0 を返す。呼び出し側は左上寄せへ倒すなどの
// 退避をとる。
func MeasureText(s string, face text.Face) (int, int) {
	if face == nil {
		return 0, 0
	}
	w, h := text.Measure(s, face, 0)
	return int(math.Round(w)), int(math.Round(h))
}

// MeasureTextWidth は MeasureText の幅だけを返す。列幅や送り幅の算出に使う。
func MeasureTextWidth(s string, face text.Face) int {
	w, _ := MeasureText(s, face)
	return w
}

// LineHeight は face の自然な行送りを画素で返す。アセンダとディセンダを含む字面の
// 高さから取るので、固定値のように字面に対して間延びしない。
func LineHeight(face text.Face) int {
	// アセンダとディセンダを両方持つ字を測り、フェイスの縦の広がりを代表させる
	_, h := MeasureText("Ag", face)
	return h
}
