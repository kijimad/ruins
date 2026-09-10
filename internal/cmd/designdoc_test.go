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

// writeDoc は dir 直下に name のドキュメントを書き込む。コアが dir を引数で受けるので、
// カレントディレクトリを触らずに済み、テストは並列化できる。
func writeDoc(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func TestDesignDocValidate_問題なしならnilを返す(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeDoc(t, dir, "ok.md", "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# OK\n")

	var buf bytes.Buffer
	require.NoError(t, designDocValidate(&buf, dir))
	assert.Equal(t, "OK: validated 1 documents\n", buf.String())
}

func TestDesignDocValidate_問題があればエラーを返す(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeDoc(t, dir, "bad.md", "---\nstatus: bogus\ntags: []\nauto: needs-decision\n---\n\n# Bad\n")

	var buf bytes.Buffer
	err := designDocValidate(&buf, dir)
	require.ErrorIs(t, err, errValidation)
	assert.Equal(t, filepath.Join(dir, "bad.md")+": invalid status: \"bogus\"\n", buf.String())
}

func TestDesignDocValidate_ディレクトリが無ければエラーを返す(t *testing.T) {
	t.Parallel()
	err := designDocValidate(io.Discard, filepath.Join(t.TempDir(), "nonexistent"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestDesignDocGen_frontmatterがないドキュメントに既定値を付与する(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeDoc(t, dir, "nofront.md", "# タイトル\n\n本文\n")

	var buf bytes.Buffer
	require.NoError(t, designDocGen(&buf, dir))
	assert.Equal(t, "added: "+filepath.Join(dir, "nofront.md")+"\nadded frontmatter to 1 documents\n", buf.String())

	got, err := os.ReadFile(filepath.Join(dir, "nofront.md"))
	require.NoError(t, err)
	assert.Equal(t, "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# タイトル\n\n本文\n", string(got))
}

func TestDesignDocGen_frontmatterがあるドキュメントは変更しない(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# 既にある\n"
	writeDoc(t, dir, "hasfront.md", content)

	var buf bytes.Buffer
	require.NoError(t, designDocGen(&buf, dir))
	assert.Equal(t, "added frontmatter to 0 documents\n", buf.String())

	got, err := os.ReadFile(filepath.Join(dir, "hasfront.md"))
	require.NoError(t, err)
	assert.Equal(t, content, string(got))
}

func TestDesignDocList_フィルタで絞り込む(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		filter designDocFilter
		docs   map[string]string
		want   []string // 出力に含まれるべき文字列
		absent []string // 出力に含まれてはならない文字列
	}{
		{
			name:   "フィルタなしで全件表示する",
			filter: designDocFilter{},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# A\n",
			},
			want: []string{"PATH", "a.md", "draft"},
		},
		{
			name:   "statusで絞り込む",
			filter: designDocFilter{status: "draft"},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# A\n",
				"b.md": "---\nstatus: done\ntags: []\nauto: needs-decision\n---\n\n# B\n",
			},
			want:   []string{"a.md"},
			absent: []string{"b.md", "done"},
		},
		{
			name:   "autoで絞り込む",
			filter: designDocFilter{auto: "mechanical"},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: []\nauto: mechanical\n---\n\n# A\n",
				"b.md": "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# B\n",
			},
			want:   []string{"a.md", "mechanical"},
			absent: []string{"b.md"},
		},
		{
			name:   "tagで絞り込む",
			filter: designDocFilter{tag: "ci"},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: [ci]\nauto: needs-decision\n---\n\n# A\n",
				"b.md": "---\nstatus: draft\ntags: [ui]\nauto: needs-decision\n---\n\n# B\n",
			},
			want:   []string{"a.md"},
			absent: []string{"b.md", "ui"},
		},
		{
			name:   "openで未着手のみ絞り込む",
			filter: designDocFilter{open: true},
			docs: map[string]string{
				"a.md": "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# A\n",
				"b.md": "---\nstatus: done\ntags: []\nauto: needs-decision\n---\n\n# B\n",
			},
			want:   []string{"a.md"},
			absent: []string{"b.md", "done"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for name, content := range tt.docs {
				writeDoc(t, dir, name, content)
			}

			var buf bytes.Buffer
			require.NoError(t, designDocList(&buf, dir, tt.filter))
			out := buf.String()
			for _, s := range tt.want {
				assert.Contains(t, out, s)
			}
			for _, s := range tt.absent {
				assert.NotContains(t, out, s)
			}
		})
	}
}
