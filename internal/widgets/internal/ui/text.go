package ui

import (
	"strings"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// WrapText は s を maxWidth に収まる複数行へ折って返す。フェイスで幅を測る。
// 空白で単語に区切り、貪欲に詰める。1単語が幅を超えるときはその単語を単独の行にする。
// フェイスが nil か maxWidth が非正なら折らずにそのまま1行で返す。
func WrapText(s string, face text.Face, maxWidth int) []string {
	if face == nil || maxWidth <= 0 || s == "" {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, word := range words[1:] {
		candidate := cur + " " + word
		width, _ := text.Measure(candidate, face, 0)
		if int(width) <= maxWidth {
			cur = candidate
			continue
		}
		lines = append(lines, cur)
		cur = word
	}
	lines = append(lines, cur)
	return lines
}
