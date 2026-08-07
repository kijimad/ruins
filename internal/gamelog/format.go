package gamelog

import (
	"fmt"
	"image/color"
	"strings"
)

// Segment は Fmt に差し込む色付きの断片。書式の %s 位置へ順に置かれる。
type Segment struct {
	Text  string
	Color color.RGBA
}

// Plain は白の断片を作る。メンバー名など装飾しない値に使う。
func Plain(v any) Segment {
	return Segment{Text: fmt.Sprintf("%v", v), Color: ColorWhite}
}

// Item はアイテム名をシアンの断片にする。ItemName メソッドと同じ色。
func Item(v any) Segment {
	return Segment{Text: fmt.Sprintf("%v", v), Color: ColorCyan}
}

// Fmt は format の %s を segs で順に置き換えて色付きログを組む。翻訳で語順が変わっても、
// 断片の色を保ったまま差し込める。%s 以外の文字は現在色の断片になる。%s の個数と segs を一致させる。
// %s 以外の書式指定子は扱わない。
func (l *Logger) Fmt(format string, segs ...Segment) *Logger {
	parts := strings.Split(format, "%s")
	for i, part := range parts {
		if part != "" {
			l.fragments = append(l.fragments, LogFragment{Color: l.currentColor, Text: part})
		}
		if i < len(segs) {
			l.fragments = append(l.fragments, LogFragment{Color: segs[i].Color, Text: segs[i].Text})
		}
	}
	return l
}
