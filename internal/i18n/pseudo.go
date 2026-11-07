package i18n

import (
	"fmt"
	"strings"
)

// PseudoLang は疑似訳の言語コード。対応言語に含めないので設定からは選べない。
const PseudoLang = "qps"

// pseudoTranslate は msgid を疑似訳にする。書式の動詞とマークアップのタグは整形と解釈が壊れないよう残す。
func pseudoTranslate(msgid string, args ...any) string {
	var b strings.Builder
	for i := 0; i < len(msgid); i++ {
		c := msgid[i]
		switch {
		case c == '%':
			end := formatVerbEnd(msgid, i)
			b.WriteString(msgid[i:end])
			i = end - 1
		case c == '<':
			end := strings.IndexByte(msgid[i:], '>')
			if end < 0 {
				b.WriteString(msgid[i:])
				i = len(msgid)
				continue
			}
			b.WriteString(msgid[i : i+end+1])
			i += end
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z':
			b.WriteRune(rune(c) + fullwidthOffset)
		default:
			b.WriteByte(c)
		}
	}
	if len(args) == 0 {
		return b.String()
	}
	return fmt.Sprintf(b.String(), args...)
}

// fullwidthOffset は ASCII の英字から全角の英字への距離。大文字も小文字も同じ距離にある。
const fullwidthOffset = 'Ａ' - 'A'

// formatVerbEnd は start の % から始まる書式の動詞の直後の位置を返す。動詞にならない % は % だけを返す。
func formatVerbEnd(s string, start int) int {
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', c == '%':
			return i + 1
		case !strings.ContainsRune("+-# 0123456789.[]*", rune(c)):
			return start + 1
		}
	}
	return start + 1
}
