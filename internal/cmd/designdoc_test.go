package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kijimaD/ruins/internal/designdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// captureOutput はos.Stdoutの出力をキャプチャする
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// writeDesignDoc はカレントディレクトリ配下の designdoc.DefaultDir にドキュメントを1件書く。
func writeDesignDoc(t *testing.T, name, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(designdoc.DefaultDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(designdoc.DefaultDir, name), []byte(content), 0o644))
}

// runDesignDocSubcommand は CmdDesignDoc を実際の cli 解析を通して実行する。
func runDesignDocSubcommand(args ...string) error {
	app := &cli.Command{
		Name:     "ruins",
		Commands: []*cli.Command{CmdDesignDoc},
	}
	return app.Run(context.Background(), append([]string{"ruins"}, args...))
}

//nolint:paralleltest // t.Chdir はプロセス全体のcwdを変えるためt.Parallelと併用できない
func TestRunDesignDocValidate_問題なしなら件数を報告してnilを返す(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "20260101_01.md", "---\nstatus: draft\ntags: []\nauto: mechanical\n---\n\n# タイトル\n\n本文\n")

	var err error
	out := captureOutput(func() {
		err = runDesignDocSubcommand("designdoc", "validate")
	})

	require.NoError(t, err)
	assert.Contains(t, out, "OK: 1 件のドキュメントを検証した")
}

//nolint:paralleltest // t.Chdir はプロセス全体のcwdを変えるためt.Parallelと併用できない
func TestRunDesignDocValidate_不正なstatusはerrValidationを返す(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "20260101_01.md", "---\nstatus: unknown\ntags: []\nauto: mechanical\n---\n\n# タイトル\n\n本文\n")

	var err error
	out := captureOutput(func() {
		err = runDesignDocSubcommand("designdoc", "validate")
	})

	require.Error(t, err)
	require.ErrorIs(t, err, errValidation)
	assert.Contains(t, out, `status が不正: "unknown"`)
}

//nolint:paralleltest // t.Chdir はプロセス全体のcwdを変えるためt.Parallelと併用できない
func TestRunDesignDocGen_frontmatter未付与のドキュメントに既定値を付与する(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "20260101_01.md", "# タイトル\n\n本文\n")

	var err error
	out := captureOutput(func() {
		err = runDesignDocSubcommand("designdoc", "gen")
	})

	require.NoError(t, err)
	assert.Contains(t, out, "1 件に frontmatter を付与した")

	docs, loadErr := designdoc.LoadDir(designdoc.DefaultDir)
	require.NoError(t, loadErr)
	require.Len(t, docs, 1)
	assert.True(t, docs[0].HasFront)
	assert.Equal(t, designdoc.StatusDraft, docs[0].Front.Status)
}

//nolint:paralleltest // t.Chdir はプロセス全体のcwdを変えるためt.Parallelと併用できない
func TestRunDesignDocGen_既にfrontmatterがあれば変更しない(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "20260101_01.md", "---\nstatus: done\ntags: []\nauto: mechanical\n---\n\n# タイトル\n\n本文\n")

	var err error
	out := captureOutput(func() {
		err = runDesignDocSubcommand("designdoc", "gen")
	})

	require.NoError(t, err)
	assert.Contains(t, out, "0 件に frontmatter を付与した")
}

//nolint:paralleltest // t.Chdir はプロセス全体のcwdを変えるためt.Parallelと併用できない
func TestRunDesignDocList_statusで絞り込む(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "20260101_01.md", "---\nstatus: draft\ntags: []\nauto: mechanical\n---\n\n# ドラフト文書\n\n本文\n")
	writeDesignDoc(t, "20260101_02.md", "---\nstatus: done\ntags: []\nauto: mechanical\n---\n\n# 完了文書\n\n本文\n")

	var err error
	out := captureOutput(func() {
		err = runDesignDocSubcommand("designdoc", "list", "--status", "done")
	})

	require.NoError(t, err)
	assert.Contains(t, out, "20260101_02.md")
	assert.NotContains(t, out, "20260101_01.md")
}

//nolint:paralleltest // t.Chdir はプロセス全体のcwdを変えるためt.Parallelと併用できない
func TestRunDesignDocList_openで未完了のみに絞り込む(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "20260101_01.md", "---\nstatus: draft\ntags: []\nauto: mechanical\n---\n\n# ドラフト文書\n\n本文\n")
	writeDesignDoc(t, "20260101_02.md", "---\nstatus: done\ntags: []\nauto: mechanical\n---\n\n# 完了文書\n\n本文\n")

	var err error
	out := captureOutput(func() {
		err = runDesignDocSubcommand("designdoc", "list", "--open")
	})

	require.NoError(t, err)
	assert.Contains(t, out, "20260101_01.md")
	assert.NotContains(t, out, "20260101_02.md")
}
