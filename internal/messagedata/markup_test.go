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

	t.Run("意味タグは前景色のみ解決する", func(t *testing.T) {
		t.Parallel()
		style, ok := tagStyle("item")
		require.True(t, ok)
		require.NotNil(t, style.fg)
		assert.Equal(t, gamelog.ColorCyan, *style.fg)
		assert.Nil(t, style.bg, "意味タグは背景を持たない")
	})

	t.Run("keywordは前景色と背景色を両方解決する", func(t *testing.T) {
		t.Parallel()
		style, ok := tagStyle("keyword")
		require.True(t, ok)
		require.NotNil(t, style.fg)
		require.NotNil(t, style.bg)
		assert.Equal(t, color.RGBA{R: 255, G: 100, B: 100, A: 255}, *style.fg)
		assert.Equal(t, color.RGBA{R: 80, G: 20, B: 20, A: 180}, *style.bg)
	})

	t.Run("color_接頭辞は既知の色名を解決する", func(t *testing.T) {
		t.Parallel()
		style, ok := tagStyle("color_red")
		require.True(t, ok)
		require.NotNil(t, style.fg)
		assert.Equal(t, gamelog.ColorRed, *style.fg)
		assert.Nil(t, style.bg)
	})

	t.Run("color_接頭辞でも未知の色名は解決しない", func(t *testing.T) {
		t.Parallel()
		_, ok := tagStyle("color_unknown")
		assert.False(t, ok)
	})

	t.Run("接頭辞の無い未知タグは解決しない", func(t *testing.T) {
		t.Parallel()
		_, ok := tagStyle("unknown")
		assert.False(t, ok)
	})

	t.Run("color_接頭辞のみで色名が無ければ解決しない", func(t *testing.T) {
		t.Parallel()
		_, ok := tagStyle("color_")
		assert.False(t, ok)
	})
}

func TestParseMarkupSegments(t *testing.T) {
	t.Parallel()

	t.Run("閉じ括弧が無いマークは地の文として残す", func(t *testing.T) {
		t.Parallel()
		segs := parseMarkupSegments("<item タグ")
		require.Len(t, segs, 1)
		assert.Equal(t, "<item タグ", segs[0].text)
		assert.Nil(t, segs[0].style.fg)
	})

	t.Run("既知タグでも閉じタグが無ければタグ自体を地の文として残す", func(t *testing.T) {
		t.Parallel()
		segs := parseMarkupSegments("<item>薬")
		require.Len(t, segs, 2)
		assert.Equal(t, "<item>", segs[0].text)
		assert.Nil(t, segs[0].style.fg)
		assert.Equal(t, "薬", segs[1].text)
		assert.Nil(t, segs[1].style.fg)
	})

	t.Run("タグ無しは既定スタイル1断片になる", func(t *testing.T) {
		t.Parallel()
		segs := parseMarkupSegments("ただの文")
		require.Len(t, segs, 1)
		assert.Equal(t, "ただの文", segs[0].text)
		assert.Nil(t, segs[0].style.fg)
	})

	t.Run("空文字は空の断片列を返す", func(t *testing.T) {
		t.Parallel()
		segs := parseMarkupSegments("")
		assert.Empty(t, segs)
	})

	t.Run("中身の無い既知タグは断片を残さない", func(t *testing.T) {
		t.Parallel()
		segs := parseMarkupSegments("<item></item>")
		assert.Empty(t, segs)
	})

	t.Run("閉じタグが対応しないマークはすべて地の文になる", func(t *testing.T) {
		t.Parallel()
		segs := parseMarkupSegments("<item>a</npc>")
		var got strings.Builder
		for _, seg := range segs {
			assert.Nil(t, seg.style.fg, "対応しない閉じタグは地の文として残る")
			got.WriteString(seg.text)
		}
		assert.Equal(t, "<item>a</npc>", got.String())
	})

	t.Run("入れ子タグは解釈されず外側タグの中身としてそのまま残る", func(t *testing.T) {
		t.Parallel()
		segs := parseMarkupSegments("<item><keyword>x</keyword></item>")
		require.Len(t, segs, 1)
		assert.Equal(t, "<keyword>x</keyword>", segs[0].text)
		assert.NotNil(t, segs[0].style.fg, "内側のタグは解釈されず外側 item のスタイルが適用される")
	})

	t.Run("空タグは未知タグとして地の文に残る", func(t *testing.T) {
		t.Parallel()
		segs := parseMarkupSegments("<>")
		require.Len(t, segs, 1)
		assert.Equal(t, "<>", segs[0].text)
		assert.Nil(t, segs[0].style.fg)
	})
}
