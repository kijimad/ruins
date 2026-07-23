package designdoc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	write := func(name, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
	}
	write("20260102_2.md", "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# B\n")
	write("20260101_1.md", "---\nstatus: done\ntags: [ecs]\nauto: mechanical\n---\n\n# A\n")
	write("tmpl.md", "---\nstatus: draft\ntags: []\nauto: needs-decision\n---\n\n# {タイトル}\n") // 雛形は除外
	write("20260103_3.drawio.svg", "<svg/>")                                                     // .md 以外は除外

	docs, err := LoadDir(dir)
	require.NoError(t, err)
	require.Len(t, docs, 2)

	// ファイル名昇順に並ぶ。
	assert.Equal(t, filepath.Join(dir, "20260101_1.md"), docs[0].Path)
	assert.Equal(t, "A", docs[0].Title)
	assert.Equal(t, 1, docs[0].Number)
	assert.Equal(t, StatusDone, docs[0].Front.Status)
	assert.Equal(t, "B", docs[1].Title)
	assert.Equal(t, 2, docs[1].Number)
}

func TestLoadDir_Error(t *testing.T) {
	t.Parallel()

	_, err := LoadDir(filepath.Join(t.TempDir(), "存在しない"))
	require.Error(t, err)
}

func TestBackfillDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "20260101_1.md")
	require.NoError(t, os.WriteFile(path, []byte("# タイトル\n\n## 進捗\n\n- [x] a\n- [ ] b\n"), 0644))
	// 雛形は付与対象外。
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tmpl.md"), []byte("# {タイトル}\n"), 0644))

	changed, err := BackfillDir(dir)
	require.NoError(t, err)
	require.Equal(t, []string{path}, changed)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	doc, err := Parse(path, string(content))
	require.NoError(t, err)
	assert.True(t, doc.HasFront)
	assert.Equal(t, StatusInProgress, doc.Front.Status) // 未完タスクありなので in-progress

	// 2回目は付与済みなので何も変えない。
	changed2, err := BackfillDir(dir)
	require.NoError(t, err)
	assert.Empty(t, changed2)
}

func TestBackfillDir_Error(t *testing.T) {
	t.Parallel()

	_, err := BackfillDir(filepath.Join(t.TempDir(), "存在しない"))
	require.Error(t, err)
}

// malformedDir は閉じデリミタの無い壊れたドキュメントを1つ置いたディレクトリを返す。
func malformedDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "20260101_1.md"), []byte("---\nstatus: draft\n# 閉じデリミタなし\n"), 0644))

	return dir
}

func TestLoadDir_MalformedDoc(t *testing.T) {
	t.Parallel()

	_, err := LoadDir(malformedDir(t))
	require.Error(t, err)
}

func TestBackfillDir_MalformedDoc(t *testing.T) {
	t.Parallel()

	_, err := BackfillDir(malformedDir(t))
	require.Error(t, err)
}

func TestSeverityString(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ERROR", SeverityError.String())
	assert.Equal(t, "WARN", SeverityWarn.String())
	assert.Equal(t, "UNKNOWN", Severity(99).String())
}

func TestHasError(t *testing.T) {
	t.Parallel()

	assert.False(t, HasError(nil))
	assert.False(t, HasError([]Problem{{Severity: SeverityWarn}}))
	assert.True(t, HasError([]Problem{{Severity: SeverityWarn}, {Severity: SeverityError}}))
}
