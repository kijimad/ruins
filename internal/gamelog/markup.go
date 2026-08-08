package gamelog

import (
	"fmt"
	"image/color"
	"strings"
)

// tagColors はマークアップのセマンティックタグを色へ対応させる。
// 翻訳文字列に <item>薬</item> のように書くと、その範囲がこの色で描かれる。
// 色は文字列の中にあるので、訳語ごと語順とともに動かせる
var tagColors = map[string]color.RGBA{
	"item":     ColorCyan,
	"player":   ColorGreen,
	"npc":      ColorYellow,
	"damage":   ColorRed,
	"success":  ColorGreen,
	"warning":  ColorYellow,
	"error":    ColorRed,
	"location": ColorOrange,
	"action":   ColorPurple,
	"money":    ColorYellow,
	"system":   ColorCyan,
}

// namedColors は <color_red> のように色名で直接指定するタグ用のパレット
var namedColors = map[string]color.RGBA{
	"white":   ColorWhite,
	"red":     ColorRed,
	"green":   ColorGreen,
	"blue":    ColorBlue,
	"yellow":  ColorYellow,
	"cyan":    ColorCyan,
	"magenta": ColorMagenta,
	"orange":  ColorOrange,
	"purple":  ColorPurple,
}

// Markup は <tag>...</tag> マークアップ付き文字列を色付き断片へ解釈してログへ積む。
// マークアップは開発者が書く信頼できる入力なので、未知タグや閉じ忘れは記述ミスとして panic する
func (l *Logger) Markup(s string) *Logger {
	l.fragments = append(l.fragments, ParseMarkup(s)...)
	return l
}

// ParseMarkup は <tag>...</tag> を色付き断片列へ分解する。タグの外は白。
// 未知タグ・閉じ括弧の無いマーク・閉じタグ抜けは記述ミスとして panic し、黙って地の文に落とさない。
func ParseMarkup(s string) []LogFragment {
	var frags []LogFragment
	add := func(text string, c color.RGBA) {
		if text != "" {
			frags = append(frags, LogFragment{Color: c, Text: text})
		}
	}
	for len(s) > 0 {
		open := strings.IndexByte(s, '<')
		if open < 0 {
			add(s, ColorWhite)
			break
		}
		add(s[:open], ColorWhite)
		s = s[open:]

		gt := strings.IndexByte(s, '>')
		if gt < 0 {
			panic(fmt.Sprintf("gamelog: unclosed markup bracket in %q", s))
		}
		tag := s[1:gt]
		c, known := tagColor(tag)
		if !known {
			panic(fmt.Sprintf("gamelog: unknown markup tag <%s>", tag))
		}
		closeTag := "</" + tag + ">"
		ci := strings.Index(s[gt+1:], closeTag)
		if ci < 0 {
			panic(fmt.Sprintf("gamelog: missing close tag %s", closeTag))
		}
		add(s[gt+1:gt+1+ci], c)
		s = s[gt+1+ci+len(closeTag):]
	}
	return frags
}

// tagColor はタグ名を色へ解決する。セマンティックタグと color_XXX 形式に対応する
func tagColor(tag string) (color.RGBA, bool) {
	if c, ok := tagColors[tag]; ok {
		return c, true
	}
	if named, ok := strings.CutPrefix(tag, "color_"); ok {
		if c, ok := namedColors[named]; ok {
			return c, true
		}
	}
	return color.RGBA{}, false
}

// Tag は text を tag で包んだマークアップ文字列を返す。色付きの値を書式の引数として
// 渡したいときに使う。例: query.T(world, "%[1]s を得た", gamelog.Tag("item", name))
func Tag(tag, text string) string {
	return fmt.Sprintf("<%s>%s</%s>", tag, text, tag)
}
