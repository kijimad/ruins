package designdoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderStatusSection(t *testing.T) {
	t.Parallel()

	docs := []*Document{
		{Path: "docs/design/260715123045.md", Title: "走り", Front: Frontmatter{Status: StatusInProgress}, HasProgress: true, DoneTasks: 13, OpenTasks: 13},
		{Path: "docs/design/260710091500.md", Title: "Ark移行", Front: Frontmatter{Status: StatusInProgress, Tags: []string{"ecs"}}, HasProgress: true, DoneTasks: 30, OpenTasks: 3, SkippedTasks: 2},
		{Path: "docs/design/260122084500.md", Title: "下書き", Front: Frontmatter{Status: StatusDraft}},
		{Path: "docs/design/260626133000.md", Title: "完了", Front: Frontmatter{Status: StatusDone}, HasProgress: true, DoneTasks: 5},
	}
	out := RenderStatusSection(docs)

	// 件数サマリは出さない。done が増えるたびに変わりブランチ間で衝突するため。
	assert.NotContains(t, out, "件数")

	// 未完了リストは status・タイトルリンク・進捗・tags を並べる。No. 列は持たない。
	assert.Contains(t, out, "| in-progress | [走り](docs/design/260715123045.md) | 13/26 |  |")
	// 見送りは分母から外し、別表記で添える。
	assert.Contains(t, out, "| in-progress | [Ark移行](docs/design/260710091500.md) | 30/33（見送り2） | ecs |")
	// draft も未完了リストに出す。
	assert.Contains(t, out, "| draft | [下書き](docs/design/260122084500.md) | - |  |")
	// done は未完了リストに出さない。
	assert.NotContains(t, out, "完了")
}

func TestRenderStatusSection_未完了なし(t *testing.T) {
	t.Parallel()

	docs := []*Document{{Title: "完了", Front: Frontmatter{Status: StatusDone}}}
	out := RenderStatusSection(docs)
	assert.Contains(t, out, "未完了のドキュメントなし")
}

func TestTitleCell(t *testing.T) {
	t.Parallel()

	// タイトルはファイルへのリンクにする。
	assert.Equal(t, "[タイトル](docs/design/x.md)", titleCell(&Document{Title: "タイトル", Path: "docs/design/x.md"}))
	// タイトルが空ならパスをラベルにしてリンクする。
	assert.Equal(t, "[docs/design/x.md](docs/design/x.md)", titleCell(&Document{Title: "", Path: "docs/design/x.md"}))
	// パスも無ければ代替文字列。
	assert.Equal(t, "(タイトルなし)", titleCell(&Document{Title: "", Path: ""}))
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
