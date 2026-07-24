package designdoc

import (
	"testing"

	"github.com/sebdah/goldie/v2"
)

// assertGoldenMarkdown は markdown をゴールデンファイルと比較する。
func assertGoldenMarkdown(t *testing.T, name, actual string) {
	t.Helper()

	// エディタで markdown として開けるよう拡張子を .golden.md にする
	g := goldie.New(t, goldie.WithNameSuffix(".golden.md"))
	g.Assert(t, name, []byte(actual))
}

func TestRenderStatusSectionGolden(t *testing.T) {
	t.Parallel()

	docs := []*Document{
		{Number: 42, Path: "docs/design/20260628_42.md", Title: "隊員アイテム運搬システム",
			Front: Frontmatter{Status: StatusInProgress, Tags: []string{"member"}}, HasProgress: true, DoneTasks: 0, OpenTasks: 11},
		{Number: 51, Path: "docs/design/20260710_51.md", Title: "ECSエンジンの Ark 移行",
			Front: Frontmatter{Status: StatusInProgress, Tags: []string{"ecs", "refactor"}}, HasProgress: true, DoneTasks: 30, OpenTasks: 3, SkippedTasks: 2},
		{Number: 1, Path: "docs/design/20260122_1.md", Title: "完了した設計",
			Front: Frontmatter{Status: StatusDone}, HasProgress: true, DoneTasks: 6},
		{Number: 27, Path: "docs/design/20260609_27.md", Title: "不採用の設計",
			Front: Frontmatter{Status: StatusDropped}, HasProgress: true, SkippedTasks: 5},
	}
	assertGoldenMarkdown(t, "status_section", RenderStatusSection(docs))
}

func TestRenderStatusSectionGolden_AllDone(t *testing.T) {
	t.Parallel()

	docs := []*Document{
		{Number: 1, Path: "docs/design/20260122_1.md", Title: "設計A", Front: Frontmatter{Status: StatusDone}, HasProgress: true, DoneTasks: 3},
		{Number: 2, Path: "docs/design/20260123_2.md", Title: "設計B", Front: Frontmatter{Status: StatusDone}, HasProgress: true, DoneTasks: 5},
	}
	assertGoldenMarkdown(t, "status_section_all_done", RenderStatusSection(docs))
}
