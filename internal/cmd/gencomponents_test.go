package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateComponents_Golden は登録表からの生成結果を、ゴールデンである
// コミット済み components_gen.go と突き合わせる。テンプレートや登録表の変更が
// 生成物に反映されていることをローカルの make test で確認し、再生成漏れを検出する。
// generateComponents は format.Source を通すため、Goとして妥当でなければ require.NoError で落ちる。
func TestGenerateComponents_Golden(t *testing.T) {
	t.Parallel()

	got, err := generateComponents()
	require.NoError(t, err)

	want, err := os.ReadFile("../components/components_gen.go")
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got),
		"components_gen.go が登録表と一致しない。`make generate` を実行すること")
}

func TestGenComponents_出力ファイルに生成コードを書き込む(t *testing.T) {
	t.Parallel()

	outPath := filepath.Join(t.TempDir(), "components_gen.go")

	var buf bytes.Buffer
	require.NoError(t, genComponents(&buf, outPath))
	assert.Equal(t, "Generated "+outPath+"\n", buf.String())

	got, err := os.ReadFile(outPath)
	require.NoError(t, err)

	want, err := generateComponents()
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

func TestGenComponents_書き込み失敗時はエラーを返す(t *testing.T) {
	t.Parallel()

	// 存在しないディレクトリへの書き込みを指定してos.WriteFileを失敗させる
	outPath := filepath.Join(t.TempDir(), "no-such-dir", "components_gen.go")

	err := genComponents(io.Discard, outPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to write generated code")
}
