package messagedata

import (
	"image/color"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddMarkup(t *testing.T) {
	t.Parallel()

	t.Run("keyword は赤文字と背景を持つ", func(t *testing.T) {
		t.Parallel()
		m := (&MessageData{}).AddMarkup("あ<keyword>強調</keyword>い")
		require.Len(t, m.TextSegmentLines, 1)
		segs := m.TextSegmentLines[0]
		require.Len(t, segs, 3)
		assert.Equal(t, "あ", segs[0].Text)
		assert.Nil(t, segs[0].Color, "地の文は既定色")
		assert.Equal(t, "強調", segs[1].Text)
		require.NotNil(t, segs[1].Color)
		require.NotNil(t, segs[1].BackgroundColor, "keyword は背景色を持つ")
		assert.Equal(t, "い", segs[2].Text)
	})

	t.Run("意味タグは前景色のみ", func(t *testing.T) {
		t.Parallel()
		m := (&MessageData{}).AddMarkup("<item>薬</item>")
		seg := m.TextSegmentLines[0][0]
		require.NotNil(t, seg.Color)
		assert.Equal(t, gamelog.ColorCyan, *seg.Color)
		assert.Nil(t, seg.BackgroundColor, "意味タグは背景を持たない")
	})

	t.Run("改行で行分割する", func(t *testing.T) {
		t.Parallel()
		m := (&MessageData{}).AddMarkup("一\n<keyword>二</keyword>\n三")
		require.Len(t, m.TextSegmentLines, 3)
		// 各行の表示テキストを連結して行分割を確認する。改行境界に空断片が挟まっても表示は同じ
		lineText := func(segs []TextSegment) string {
			var s strings.Builder
			for _, seg := range segs {
				s.WriteString(seg.Text)
			}
			return s.String()
		}
		assert.Equal(t, "一", lineText(m.TextSegmentLines[0]))
		assert.Equal(t, "二", lineText(m.TextSegmentLines[1]))
		assert.Equal(t, "三", lineText(m.TextSegmentLines[2]))
		// 中央行の keyword 断片は背景色を持つ
		hasBg := false
		for _, seg := range m.TextSegmentLines[1] {
			if seg.Text == "二" && seg.BackgroundColor != nil {
				hasBg = true
			}
		}
		assert.True(t, hasBg, "keyword は背景色を持つ")
	})

	t.Run("未知タグは地の文として残す", func(t *testing.T) {
		t.Parallel()
		m := (&MessageData{}).AddMarkup("a<unknown>b</unknown>c")
		var got strings.Builder
		for _, seg := range m.TextSegmentLines[0] {
			assert.Nil(t, seg.Color)
			got.WriteString(seg.Text)
		}
		assert.Equal(t, "a<unknown>b</unknown>c", got.String())
	})
}

func TestTagStyle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tag  string
		want markupStyle
		ok   bool
	}{
		{"意味タグは前景色だけを持つ", "item", fg(gamelog.ColorCyan), true},
		{"keywordは前景色と背景色を持つ", "keyword", markupStyle{fg: &color.RGBA{R: 255, G: 100, B: 100, A: 255}, bg: &color.RGBA{R: 80, G: 20, B: 20, A: 180}}, true},
		{"color_接頭辞は既知の色名を前景色にする", "color_red", fg(gamelog.ColorRed), true},
		{"color_接頭辞でも未知の色名は解決しない", "color_unknown", markupStyle{}, false},
		{"接頭辞の無い未知タグは解決しない", "unknown", markupStyle{}, false},
		{"color_接頭辞だけで色名が無ければ解決しない", "color_", markupStyle{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := tagStyle(tt.tag)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseMarkupSegments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []markupSegment
	}{
		{"閉じ括弧が無いマークは地の文として残す", "<item タグ", []markupSegment{{text: "<item タグ"}}},
		{"既知タグでも閉じタグが無ければタグ自体を地の文として残す", "<item>薬", []markupSegment{{text: "<item>"}, {text: "薬"}}},
		{"タグ無しは既定スタイル1断片になる", "ただの文", []markupSegment{{text: "ただの文"}}},
		{"空文字は空の断片列を返す", "", nil},
		{"中身の無い既知タグは断片を残さない", "<item></item>", nil},
		{"閉じタグが対応しないマークはすべて地の文になる", "<item>a</npc>", []markupSegment{{text: "<item>"}, {text: "a"}, {text: "</npc>"}}},
		{"入れ子のタグには対応せず、外側のタグの中身を文字のまま色付けする", "<item><keyword>x</keyword></item>", []markupSegment{{text: "<keyword>x</keyword>", style: fg(gamelog.ColorCyan)}}},
		{"空タグは未知タグとして地の文に残る", "<>", []markupSegment{{text: "<>"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseMarkupSegments(tt.in))
		})
	}
}
