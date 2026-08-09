package gamelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMarkup(t *testing.T) {
	t.Parallel()

	t.Run("セマンティックタグは対応色の断片になる", func(t *testing.T) {
		t.Parallel()
		frags := ParseMarkup("<item>薬</item>を得た")
		assert.Equal(t, []LogFragment{
			{Color: ColorCyan, Text: "薬"},
			{Color: ColorWhite, Text: "を得た"},
		}, frags)
	})

	t.Run("複数タグと地の文が順に並ぶ", func(t *testing.T) {
		t.Parallel()
		frags := ParseMarkup("<player>アッシュ</player>は<npc>敵</npc>を攻撃した")
		assert.Equal(t, []LogFragment{
			{Color: ColorGreen, Text: "アッシュ"},
			{Color: ColorWhite, Text: "は"},
			{Color: ColorYellow, Text: "敵"},
			{Color: ColorWhite, Text: "を攻撃した"},
		}, frags)
	})

	t.Run("color_ 形式で色名を直接指定できる", func(t *testing.T) {
		t.Parallel()
		frags := ParseMarkup("<color_red>危険</color_red>")
		assert.Equal(t, []LogFragment{{Color: ColorRed, Text: "危険"}}, frags)
	})

	t.Run("未知タグは panic する", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { ParseMarkup("a<unknown>b</unknown>c") })
	})

	t.Run("閉じ括弧が無いマークは panic する", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { ParseMarkup("壊れた<item タグ") })
	})

	t.Run("閉じタグ抜けは panic する", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() { ParseMarkup("<item>薬") })
	})

	t.Run("タグ無しは白1断片", func(t *testing.T) {
		t.Parallel()
		frags := ParseMarkup("ただの文")
		assert.Equal(t, []LogFragment{{Color: ColorWhite, Text: "ただの文"}}, frags)
	})
}

func TestTag(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "<item>薬</item>", Tag("item", "薬"))
}

func TestLoggerMarkupLogsColoredEntry(t *testing.T) {
	t.Parallel()
	store := NewSafeSlice(10)
	New(store).Markup("<item>薬</item>を得た").Log()

	entries := store.GetRecentEntries(10)
	require.Len(t, entries, 1)
	assert.Equal(t, []LogFragment{
		{Color: ColorCyan, Text: "薬"},
		{Color: ColorWhite, Text: "を得た"},
	}, entries[0].Fragments)
}
