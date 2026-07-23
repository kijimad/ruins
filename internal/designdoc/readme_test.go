package designdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderStatusSection(t *testing.T) {
	t.Parallel()

	docs := []*Document{
		{Number: 58, Path: "docs/design/20260715_58.md", Title: "走り", Front: Frontmatter{Status: StatusInProgress}, HasProgress: true, DoneTasks: 13, OpenTasks: 13},
		{Number: 51, Path: "docs/design/20260710_51.md", Title: "Ark移行", Front: Frontmatter{Status: StatusInProgress, Tags: []string{"ecs"}}, HasProgress: true, DoneTasks: 30, OpenTasks: 3, SkippedTasks: 2},
		{Number: 1, Path: "docs/design/20260122_1.md", Title: "下書き", Front: Frontmatter{Status: StatusDraft}},
		{Number: 37, Path: "docs/design/20260626_37.md", Title: "完了", Front: Frontmatter{Status: StatusDone}, HasProgress: true, DoneTasks: 5},
	}
	out := RenderStatusSection(docs)

	// 件数サマリ。
	assert.Contains(t, out, "| in-progress | 2 |")
	assert.Contains(t, out, "| draft | 1 |")
	assert.Contains(t, out, "| done | 1 |")
	// 0件の status は出さない。
	assert.NotContains(t, out, "superseded")

	// 進行中リストは 連番リンク・タイトル・進捗・tags を並べる。
	assert.Contains(t, out, "### 進行中")
	assert.Contains(t, out, "| [58](docs/design/20260715_58.md) | 走り | 13/26 |  |")
	// 見送りは分母から外し、別表記で添える。
	assert.Contains(t, out, "| [51](docs/design/20260710_51.md) | Ark移行 | 30/33（見送り2） | ecs |")
	// draft や done は進行中リストに出さない。
	assert.NotContains(t, out, "| 下書き |")
	assert.NotContains(t, out, "| 完了 |")
}

func TestRenderStatusSection_NoInProgress(t *testing.T) {
	t.Parallel()

	docs := []*Document{{Title: "下書き", Front: Frontmatter{Status: StatusDraft}}}
	out := RenderStatusSection(docs)
	assert.Contains(t, out, "進行中のドキュメントなし")
}

func TestTitleCell(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "タイトル", titleCell(&Document{Title: "タイトル", Path: "docs/design/x.md"}))
	// タイトルが空ならパスで代替する。
	assert.Equal(t, "docs/design/x.md", titleCell(&Document{Title: "", Path: "docs/design/x.md"}))
	assert.Equal(t, "(タイトルなし)", titleCell(&Document{Title: "", Path: ""}))
}

func TestNumberCell(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "-", numberCell(&Document{Number: 0}))
	assert.Equal(t, "58", numberCell(&Document{Number: 58, Path: ""}))
	assert.Equal(t, "[58](docs/design/x.md)", numberCell(&Document{Number: 58, Path: "docs/design/x.md"}))
}

func TestProgressCell(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "-", progressCell(&Document{HasProgress: false}))
	assert.Equal(t, "3/5", progressCell(&Document{HasProgress: true, DoneTasks: 3, OpenTasks: 2}))
	// 見送りは分母から外し、括弧で添える。
	assert.Equal(t, "3/3（見送り2）", progressCell(&Document{HasProgress: true, DoneTasks: 3, OpenTasks: 0, SkippedTasks: 2}))
}

func TestParse_Title(t *testing.T) {
	t.Parallel()

	doc, err := Parse("t.md", "# これはタイトル\n\n## 背景\n\n本文\n")
	require.NoError(t, err)
	assert.Equal(t, "これはタイトル", doc.Title)
	// 見出し `##` はタイトルに拾わない。
	assert.NotEqual(t, "背景", doc.Title)
	// タイトルに改行が混入しないこと。
	assert.NotContains(t, doc.Title, "\n")
}
