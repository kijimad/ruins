package designdoc

import (
	"fmt"
	"strings"
)

// 状況テーブルの見出しと、行が無いときの行
const (
	statusHeader = "| status | Document | Progress | tags |\n|---|---|---|---|\n"
	statusEmpty  = "| | No open documents | | |\n"
)

// statusDisplayOrder は状況テーブルで status を並べる順。着手対象を上に置く。
var statusDisplayOrder = []Status{
	StatusInProgress, StatusAccepted, StatusDraft, StatusDone, StatusSuperseded, StatusDropped,
}

// RenderStatusSection は README に埋め込む未完了ドキュメントの一覧を Markdown で返す。
// docs は表示したい順に並んでいる前提。載せるのはアクションが要る status、すなわち Status.IsOpen が
// true のものに限る。status 別の件数テーブルは出さない。done が増えるたびに変わり、ブランチ間で
// 頻繁に衝突するため。
func RenderStatusSection(docs []*Document) string {
	var b strings.Builder
	b.WriteString(statusHeader)
	found := false
	for _, s := range statusDisplayOrder {
		if !s.IsOpen() {
			continue
		}
		for _, d := range docs {
			if d.Front.Status != s {
				continue
			}
			found = true
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				d.Front.Status, titleCell(d), progressCell(d), strings.Join(d.Front.Tags, ", "))
		}
	}
	if !found {
		b.WriteString(statusEmpty)
	}

	return b.String()
}

// titleCell はドキュメントのセルを返す。パスがあればタイトルをファイルへのリンクにする。
// タイトルが空なら空セルにせずパスで代替し、追跡できるようにする。
func titleCell(d *Document) string {
	label := d.Title
	if label == "" {
		label = d.Path
	}
	if label == "" {
		return "(untitled)"
	}
	if d.Path != "" {
		return fmt.Sprintf("[%s](%s)", label, d.Path)
	}

	return label
}

// progressCell は進捗のセルを返す。分母は done+open で、見送りは分母から外して別に添える。
func progressCell(d *Document) string {
	if !d.HasProgress {
		return "-"
	}

	s := fmt.Sprintf("%d/%d", d.DoneTasks, d.DoneTasks+d.OpenTasks)
	if d.SkippedTasks > 0 {
		s += fmt.Sprintf(" (%d skipped)", d.SkippedTasks)
	}

	return s
}
