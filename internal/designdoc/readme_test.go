package designdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderStatusSection(t *testing.T) {
	t.Parallel()

	docs := []*Document{
		{Title: "走り", Front: Frontmatter{Status: StatusInProgress}, HasProgress: true, DoneTasks: 13, OpenTasks: 13},
		{Title: "Ark移行", Front: Frontmatter{Status: StatusInProgress, Tags: []string{"ecs"}}, HasProgress: true, DoneTasks: 30, OpenTasks: 3},
		{Title: "下書き", Front: Frontmatter{Status: StatusDraft}},
		{Title: "完了", Front: Frontmatter{Status: StatusDone}, HasProgress: true, DoneTasks: 5},
	}
	out := RenderStatusSection(docs)

	// 件数サマリ。
	assert.Contains(t, out, "| in-progress | 2 |")
	assert.Contains(t, out, "| draft | 1 |")
	assert.Contains(t, out, "| done | 1 |")
	// 0件の status は出さない。
	assert.NotContains(t, out, "superseded")

	// 進行中リストにタイトルと進捗が並ぶ。
	assert.Contains(t, out, "### 進行中")
	assert.Contains(t, out, "| 走り | 13/26 |")
	assert.Contains(t, out, "| Ark移行 | 30/33 | ecs |")
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
