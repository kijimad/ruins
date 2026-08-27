package ui

import (
	"strings"

	"github.com/go-text/typesetting/segmenter"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// WrapText は s を maxWidth に収まる複数行へ折って返す。フェイスで幅を測る。
// 折り返し位置は UAX#14 の行分割規則で求める。text/v2 と同じ go-text の segmenter へ
// 委譲するので、空白で区切られた英語も、空白の無い日本語も、同じ規則で正しく折り返せる。
// 分割候補を貪欲に詰め、行末の空白は測定と出力の両方から落とす。改行文字は強制改行にする。
// 1区切りが幅を超えるときはその区切りを単独の行にする。
// フェイスが nil か maxWidth が非正なら折らずにそのまま1行で返す。
func WrapText(s string, face text.Face, maxWidth int) []string {
	if face == nil || maxWidth <= 0 || s == "" {
		return []string{s}
	}
	var seg segmenter.Segmenter
	seg.Init([]rune(s))

	var lines []string
	cur := ""
	iter := seg.LineIterator()
	for iter.Next() {
		line := iter.Line()
		chunk := string(line.Text)
		candidate := cur + chunk
		if cur != "" {
			width, _ := text.Measure(strings.TrimRight(candidate, " "), face, 0)
			if int(width) > maxWidth {
				lines = append(lines, strings.TrimRight(cur, " "))
				candidate = chunk
			}
		}
		cur = candidate
		if line.IsMandatoryBreak {
			lines = append(lines, strings.TrimRight(cur, "\n "))
			cur = ""
		}
	}
	if cur != "" || len(lines) == 0 {
		lines = append(lines, strings.TrimRight(cur, " "))
	}
	return lines
}
