package designdoc

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Frontmatter(t *testing.T) {
	t.Parallel()

	content := "---\nstatus: in-progress\ntags: [refactor, ci]\nauto: mechanical\n---\n\n# タイトル\n\n## 進捗\n\n- [x] やった\n- [ ] まだ\n"
	doc, err := Parse("t.md", content)
	require.NoError(t, err)

	assert.True(t, doc.HasFront)
	assert.Equal(t, StatusInProgress, doc.Front.Status)
	assert.Equal(t, []string{"refactor", "ci"}, doc.Front.Tags)
	assert.Equal(t, AutoMechanical, doc.Front.Auto)
	assert.True(t, doc.HasProgress)
	assert.Equal(t, 1, doc.DoneTasks)
	assert.Equal(t, 1, doc.OpenTasks)
	assert.True(t, strings.HasPrefix(doc.Body, "# タイトル"))
}

func TestParse_NoFrontmatter(t *testing.T) {
	t.Parallel()

	content := "# タイトル\n\n## 背景\n"
	doc, err := Parse("t.md", content)
	require.NoError(t, err)

	assert.False(t, doc.HasFront)
	assert.False(t, doc.HasProgress)
	assert.Equal(t, content, doc.Body)
}

func TestParse_ProgressScopedToSection(t *testing.T) {
	t.Parallel()

	// 進捗セクションの外にあるチェックボックスは数えない。
	content := "# t\n\n## 設計\n\n- [ ] これは設計内の例\n\n## 進捗\n\n- [x] 完\n\n## コメント\n\n- [ ] これはコメント\n"
	doc, err := Parse("t.md", content)
	require.NoError(t, err)

	assert.True(t, doc.HasProgress)
	assert.Equal(t, 1, doc.DoneTasks)
	assert.Equal(t, 0, doc.OpenTasks)
}

func TestInferStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doc  *Document
		want Status
	}{
		{"進捗なし", &Document{HasProgress: false}, StatusDraft},
		{"未完あり", &Document{HasProgress: true, OpenTasks: 2, DoneTasks: 1}, StatusInProgress},
		{"全完了", &Document{HasProgress: true, OpenTasks: 0, DoneTasks: 5}, StatusDone},
		{"進捗0件", &Document{HasProgress: true, OpenTasks: 0, DoneTasks: 0}, StatusDraft},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, InferStatus(tt.doc))
		})
	}
}

func TestBackfill_Idempotent(t *testing.T) {
	t.Parallel()

	orig := "# タイトル\n\n## 進捗\n\n- [x] a\n- [ ] b\n"
	first, changed, err := Backfill(orig)
	require.NoError(t, err)
	require.True(t, changed)
	assert.Contains(t, first, "status: in-progress")
	assert.Contains(t, first, "auto: needs-decision")
	assert.Contains(t, first, "tags: []")

	// 2回目は変化しない。
	second, changed2, err := Backfill(first)
	require.NoError(t, err)
	assert.False(t, changed2)
	assert.Equal(t, first, second)
}

func TestBackfill_PreservesBody(t *testing.T) {
	t.Parallel()

	orig := "# タイトル\n\n本文\n"
	result, changed, err := Backfill(orig)
	require.NoError(t, err)
	require.True(t, changed)

	doc, err := Parse("t.md", result)
	require.NoError(t, err)
	assert.Equal(t, orig, doc.Body)
}

func TestValidate(t *testing.T) {
	t.Parallel()

	docs := []*Document{
		{Path: "no-front.md", HasFront: false},
		{Path: "bad-status.md", HasFront: true, Front: Frontmatter{Status: "wat", Auto: AutoMechanical}},
		{Path: "bad-auto.md", HasFront: true, Front: Frontmatter{Status: StatusDraft, Auto: "wat"}},
		{Path: "unknown-tag.md", HasFront: true, Front: Frontmatter{Status: StatusDraft, Auto: AutoMechanical, Tags: []string{"nope"}}},
		{Path: "done-open.md", HasFront: true, Front: Frontmatter{Status: StatusDone, Auto: AutoMechanical}, HasProgress: true, OpenTasks: 1},
		{Path: "ok.md", HasFront: true, Front: Frontmatter{Status: StatusDraft, Auto: AutoNeedsDecision, Tags: []string{"refactor"}}},
	}
	problems := Validate(docs)

	assert.True(t, HasError(problems))
	assert.Equal(t, SeverityError, findProblem(t, problems, "no-front.md").Severity)
	assert.Equal(t, SeverityError, findProblem(t, problems, "bad-status.md").Severity)
	assert.Equal(t, SeverityError, findProblem(t, problems, "bad-auto.md").Severity)
	assert.Equal(t, SeverityWarn, findProblem(t, problems, "unknown-tag.md").Severity)
	// done なのに未チェックが残るのは不変条件違反。Error で弾く。
	assert.Equal(t, SeverityError, findProblem(t, problems, "done-open.md").Severity)

	// ok.md は問題を出さない。
	for _, p := range problems {
		assert.NotEqual(t, "ok.md", p.Path)
	}
}

func TestParse_NumberAndSkip(t *testing.T) {
	t.Parallel()

	content := "# t\n\n## 進捗\n\n- [x] 済\n- [ ] 未\n- [~] 見送り\n"
	doc, err := Parse("docs/design/20260715_58.md", content)
	require.NoError(t, err)

	assert.Equal(t, 58, doc.Number)
	assert.Equal(t, 1, doc.DoneTasks)
	assert.Equal(t, 1, doc.OpenTasks)
	assert.Equal(t, 1, doc.SkippedTasks)
}

func TestStatusIsOpen(t *testing.T) {
	t.Parallel()

	assert.True(t, StatusDraft.IsOpen())
	assert.True(t, StatusAccepted.IsOpen())
	assert.True(t, StatusInProgress.IsOpen())
	assert.False(t, StatusDone.IsOpen())
	assert.False(t, StatusSuperseded.IsOpen())
	assert.False(t, StatusDropped.IsOpen())
}

// findProblem は指定パスの最初の問題を返す。無ければテストを落とす。
func findProblem(t *testing.T, problems []Problem, path string) Problem {
	t.Helper()
	for _, p := range problems {
		if p.Path == path {
			return p
		}
	}
	require.Failf(t, "問題が見つからない", "path=%s", path)

	return Problem{}
}
