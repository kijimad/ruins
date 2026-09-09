package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

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

// writeDesignDoc は tempDir/docs/design/name にドキュメントを書き込む
func writeDesignDoc(t *testing.T, name, content string) {
	t.Helper()
	dir := filepath.Join("docs", "design")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

//nolint:paralleltest // t.Chdirとos.Stdoutのキャプチャがプロセス全体を変更するため並列化しない
func TestRunDesignDocValidate_問題なしならnilを返す(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "ok.md", "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# OK\n")

	out := captureOutput(func() {
		err := runDesignDocValidate(context.Background(), nil)
		require.NoError(t, err)
	})
	assert.Equal(t, "OK: validated 1 documents\n", out)
}

//nolint:paralleltest // t.Chdirとos.Stdoutのキャプチャがプロセス全体を変更するため並列化しない
func TestRunDesignDocValidate_問題があればエラーを返す(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "bad.md", "---\nstatus: bogus\ntags: []\nauto: needs-decision\n---\n\n# Bad\n")

	var err error
	out := captureOutput(func() {
		err = runDesignDocValidate(context.Background(), nil)
	})
	require.ErrorIs(t, err, errValidation)
	assert.Equal(t, "docs/design/bad.md: invalid status: \"bogus\"\n", out)
}

//nolint:paralleltest // t.Chdirとos.Stdoutのキャプチャがプロセス全体を変更するため並列化しない
func TestRunDesignDocGen_frontmatterがないドキュメントに既定値を付与する(t *testing.T) {
	t.Chdir(t.TempDir())
	writeDesignDoc(t, "nofront.md", "# タイトル\n\n本文\n")

	out := captureOutput(func() {
		err := runDesignDocGen(context.Background(), nil)
		require.NoError(t, err)
	})
	assert.Equal(t, "added: docs/design/nofront.md\nadded frontmatter to 1 documents\n", out)

	got, err := os.ReadFile(filepath.Join("docs", "design", "nofront.md"))
	require.NoError(t, err)
	assert.Equal(t, "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# タイトル\n\n本文\n", string(got))
}

//nolint:paralleltest // t.Chdirとos.Stdoutのキャプチャがプロセス全体を変更するため並列化しない
func TestRunDesignDocGen_frontmatterがあるドキュメントは変更しない(t *testing.T) {
	t.Chdir(t.TempDir())
	content := "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# 既にある\n"
	writeDesignDoc(t, "hasfront.md", content)

	out := captureOutput(func() {
		err := runDesignDocGen(context.Background(), nil)
		require.NoError(t, err)
	})
	assert.Equal(t, "added frontmatter to 0 documents\n", out)

	got, err := os.ReadFile(filepath.Join("docs", "design", "hasfront.md"))
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
}

//nolint:paralleltest // t.Chdirがプロセス全体のカレントディレクトリを変更するため並列化しない
func TestRunDesignDocValidate_ディレクトリが無ければエラーを返す(t *testing.T) {
	t.Chdir(t.TempDir())

	err := runDesignDocValidate(context.Background(), nil)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

// newDesignDocListCmd は runDesignDocList をテストするための単体の *cli.Command を組み立てる。
// CmdDesignDoc.Commands のフラグを共有すると並行テスト間で値が競合するため、フラグ定義だけ写す
func newDesignDocListCmd() *cli.Command {
	return &cli.Command{
		Name: "list",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "status"},
			&cli.StringFlag{Name: "auto"},
			&cli.StringFlag{Name: "tag"},
			&cli.BoolFlag{Name: "open"},
		},
		Action: runDesignDocList,
	}
}

//nolint:paralleltest // t.Chdirとos.Stdoutのキャプチャがプロセス全体を変更するため並列化しない
func TestRunDesignDocList(t *testing.T) {
	tests := []struct {
		name string
		args []string
		docs map[string]string
		want string
	}{
		{
			name: "フィルタなしで全件表示する",
			args: []string{"list"},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# A\n",
			},
			want: "PATH              STATUS  AUTO            PROGRESS  TAGS\n" +
				"docs/design/a.md  draft   needs-decision  -         \n",
		},
		{
			name: "statusで絞り込む",
			args: []string{"list", "--status", "draft"},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# A\n",
				"b.md": "---\nstatus: done\ntags: []\nauto: needs-decision\n---\n\n# B\n",
			},
			want: "PATH              STATUS  AUTO            PROGRESS  TAGS\n" +
				"docs/design/a.md  draft   needs-decision  -         \n",
		},
		{
			name: "autoで絞り込む",
			args: []string{"list", "--auto", "mechanical"},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: []\nauto: mechanical\n---\n\n# A\n",
				"b.md": "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# B\n",
			},
			want: "PATH              STATUS  AUTO        PROGRESS  TAGS\n" +
				"docs/design/a.md  draft   mechanical  -         \n",
		},
		{
			name: "tagで絞り込む",
			args: []string{"list", "--tag", "ci"},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: [ci]\nauto: needs-decision\n---\n\n# A\n",
				"b.md": "---\nstatus: draft\ntags: [ui]\nauto: needs-decision\n---\n\n# B\n",
			},
			want: "PATH              STATUS  AUTO            PROGRESS  TAGS\n" +
				"docs/design/a.md  draft   needs-decision  -         ci\n",
		},
		{
			name: "openで未着手のみ絞り込む",
			args: []string{"list", "--open"},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# A\n",
				"b.md": "---\nstatus: done\ntags: []\nauto: needs-decision\n---\n\n# B\n",
			},
			want: "PATH              STATUS  AUTO            PROGRESS  TAGS\n" +
				"docs/design/a.md  draft   needs-decision  -         \n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			for name, content := range tt.docs {
				writeDesignDoc(t, name, content)
			}

			var err error
			out := captureOutput(func() {
				err = newDesignDocListCmd().Run(context.Background(), tt.args)
			})
			require.NoError(t, err)
			assert.Equal(t, tt.want, out)
		})
	}
}
