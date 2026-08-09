package messagedata

import (
	"image/color"
	"strings"

	"github.com/kijimaD/ruins/internal/gamelog"
)

// markupStyle はマークアップタグの前景色と背景色。背景色は keyword のみ持つ
type markupStyle struct {
	fg *color.RGBA
	bg *color.RGBA
}

func fg(c color.RGBA) markupStyle { return markupStyle{fg: &c} }

// tagStyles はメッセージ用マークアップのタグをスタイルへ対応させる。
// 前景色は gamelog と同じパレットで揃え、keyword だけ赤文字＋暗赤背景で強調する
var tagStyles = map[string]markupStyle{
	"keyword":  {fg: &color.RGBA{R: 255, G: 100, B: 100, A: 255}, bg: &color.RGBA{R: 80, G: 20, B: 20, A: 180}},
	"item":     fg(gamelog.ColorCyan),
	"player":   fg(gamelog.ColorGreen),
	"npc":      fg(gamelog.ColorYellow),
	"damage":   fg(gamelog.ColorRed),
	"success":  fg(gamelog.ColorGreen),
	"warning":  fg(gamelog.ColorYellow),
	"error":    fg(gamelog.ColorRed),
	"location": fg(gamelog.ColorOrange),
	"system":   fg(gamelog.ColorCyan),
}

var namedColors = map[string]color.RGBA{
	"white": gamelog.ColorWhite, "red": gamelog.ColorRed, "green": gamelog.ColorGreen,
	"blue": gamelog.ColorBlue, "yellow": gamelog.ColorYellow, "cyan": gamelog.ColorCyan,
	"magenta": gamelog.ColorMagenta, "orange": gamelog.ColorOrange, "purple": gamelog.ColorPurple,
}

func tagStyle(tag string) (markupStyle, bool) {
	if s, ok := tagStyles[tag]; ok {
		return s, true
	}
	if name, ok := strings.CutPrefix(tag, "color_"); ok {
		if c, ok := namedColors[name]; ok {
			return fg(c), true
		}
	}
	return markupStyle{}, false
}

// AddMarkup は <tag>...</tag> マークアップ付きテキストを追加する。
// タグの外は既定色、keyword は赤文字＋背景、意味タグは前景色になる。
// 改行 \n は行分割する。未知タグや閉じ忘れは literal として本文に残す
func (m *MessageData) AddMarkup(text string) *MessageData {
	m.ensureCurrentLine()
	for _, seg := range parseMarkupSegments(text) {
		// 断片ごとに \n で行を割る。AddText と同じく空文字の断片も落とさない
		lines := strings.Split(seg.text, "\n")
		for i, line := range lines {
			if i > 0 {
				m.TextSegmentLines = append(m.TextSegmentLines, []TextSegment{})
			}
			idx := len(m.TextSegmentLines) - 1
			m.TextSegmentLines[idx] = append(m.TextSegmentLines[idx], TextSegment{
				Text:            line,
				Color:           seg.style.fg,
				BackgroundColor: seg.style.bg,
			})
		}
	}
	return m
}

type markupSegment struct {
	text  string
	style markupStyle
}

// parseMarkupSegments は <tag>...</tag> をスタイル付き断片へ分解する。
// タグの外は既定スタイル、対応しないタグや閉じ括弧の無いマークは literal として残す
func parseMarkupSegments(s string) []markupSegment {
	var segs []markupSegment
	add := func(text string, style markupStyle) {
		if text != "" {
			segs = append(segs, markupSegment{text: text, style: style})
		}
	}
	for len(s) > 0 {
		open := strings.IndexByte(s, '<')
		if open < 0 {
			add(s, markupStyle{})
			break
		}
		add(s[:open], markupStyle{})
		s = s[open:]

		gt := strings.IndexByte(s, '>')
		if gt < 0 {
			add(s, markupStyle{})
			break
		}
		tag := s[1:gt]
		style, known := tagStyle(tag)
		closeTag := "</" + tag + ">"
		ci := strings.Index(s[gt+1:], closeTag)
		if !known || ci < 0 {
			add(s[:gt+1], markupStyle{})
			s = s[gt+1:]
			continue
		}
		add(s[gt+1:gt+1+ci], style)
		s = s[gt+1+ci+len(closeTag):]
	}
	return segs
}
