package uicore_test

import (
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WrapText は UAX#14 の行分割へ委譲する。実フォントで測り、言語ごとの規則を確かめる。

func TestWrapText_日本語は空白が無くても折り返す(t *testing.T) {
	t.Parallel()
	res := borrowRes()
	defer facePool.Put(res)

	s := "長い説明文は空白が無くても幅に合わせて複数の行へ折り返される"
	lines := uicore.WrapText(s, res.Text.BodyFace, 200)
	require.Greater(t, len(lines), 1, "空白の無い日本語も折り返される")
	assert.Equal(t, s, strings.Join(lines, ""), "折り返しで文字が失われない")
}

func TestWrapText_英語は空白で折り返し単語を分断しない(t *testing.T) {
	t.Parallel()
	res := borrowRes()
	defer facePool.Put(res)

	s := "A hard biscuit that keeps well as a preserved food."
	lines := uicore.WrapText(s, res.Text.BodyFace, 120)
	require.Greater(t, len(lines), 1, "幅を超える英文は折り返される")
	// 行末の空白は落ちるので、空白1つで繋ぎ直すと原文に戻る。単語の途中では切れない
	assert.Equal(t, s, strings.Join(lines, " "), "空白の境界だけで切れる")
}

func TestWrapText_改行文字は強制改行になる(t *testing.T) {
	t.Parallel()
	res := borrowRes()
	defer facePool.Put(res)

	lines := uicore.WrapText("一行目\n二行目", res.Text.BodyFace, 10000)
	assert.Equal(t, []string{"一行目", "二行目"}, lines, "幅が十分でも改行文字で行が分かれる")
}
