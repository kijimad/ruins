package gamelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_Fmt_色付き断片を語順どおり差し込む(t *testing.T) {
	t.Parallel()
	l := New(NewSafeSlice(10)).Fmt("%s は %s を外した。", Plain("アッシュ"), Item("鉄の剣"))

	require.Len(t, l.fragments, 4)
	assert.Equal(t, "アッシュ", l.fragments[0].Text)
	assert.Equal(t, ColorWhite, l.fragments[0].Color)
	assert.Equal(t, " は ", l.fragments[1].Text)
	assert.Equal(t, ColorWhite, l.fragments[1].Color)
	assert.Equal(t, "鉄の剣", l.fragments[2].Text)
	assert.Equal(t, ColorCyan, l.fragments[2].Color, "アイテム名はシアンを保つ")
	assert.Equal(t, " を外した。", l.fragments[3].Text)
}

func TestLogger_Fmt_先頭が差し込みでも順序を保つ(t *testing.T) {
	t.Parallel()
	l := New(NewSafeSlice(10)).Fmt("%sを構えた", Item("レイガン"))

	require.Len(t, l.fragments, 2)
	assert.Equal(t, "レイガン", l.fragments[0].Text)
	assert.Equal(t, ColorCyan, l.fragments[0].Color)
	assert.Equal(t, "を構えた", l.fragments[1].Text)
	assert.Equal(t, ColorWhite, l.fragments[1].Color)
}
