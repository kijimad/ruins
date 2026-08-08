package messagedata

import (
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
