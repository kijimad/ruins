package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratedFileIsUpToDate は生成器の出力とコミット済み components_gen.go の一致を検証する。
// 登録表を変更したのに `make generate` を忘れた場合、この規約テストが検出する。
func TestGeneratedFileIsUpToDate(t *testing.T) {
	t.Parallel()

	const path = "../components_gen.go"

	want, err := generate()
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got),
		"components_gen.go が登録表と一致しない。`make generate` を実行すること")
}
