package designdoc

import (
	"fmt"
	"strings"
)

// statusDisplayOrder は状況テーブルで status を並べる順。着手対象を上に置く。
var statusDisplayOrder = []Status{
	StatusInProgress, StatusAccepted, StatusDraft, StatusDone, StatusSuperseded, StatusDropped,
}

// RenderStatusSection は README に埋め込む状況テーブルを Markdown で返す。
// status 別の件数と、進行中ドキュメントの一覧を出す。docs は表示したい順に並んでいる前提。
func RenderStatusSection(docs []*Document) string {
	counts := map[Status]int{}
	for _, d := range docs {
		counts[d.Front.Status]++
	}

	var b strings.Builder
	b.WriteString("| status | 件数 |\n|---|---|\n")
	for _, s := range statusDisplayOrder {
		if counts[s] == 0 {
			continue
		}
		fmt.Fprintf(&b, "| %s | %d |\n", s, counts[s])
	}

	b.WriteString("\n### 進行中\n\n")
	b.WriteString("| ドキュメント | 進捗 | tags |\n|---|---|---|\n")
	found := false
	for _, d := range docs {
		if d.Front.Status != StatusInProgress {
			continue
		}
		found = true
		progress := "-"
		if d.HasProgress {
			progress = fmt.Sprintf("%d/%d", d.DoneTasks, d.DoneTasks+d.OpenTasks)
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", d.Title, progress, strings.Join(d.Front.Tags, ", "))
	}
	if !found {
		b.WriteString("| 進行中のドキュメントなし |  |  |\n")
	}

	return b.String()
}
