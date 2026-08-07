package designdoc

import (
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

// statusSectionTmplSrc は状況テーブルの日本語ラベルを持つ雛形。
// 日本語ドキュメントの体裁を維持するため、リテラルを Go ソースから外して埋め込みで持つ。
//
//go:embed status_section.tmpl
var statusSectionTmplSrc string

// statusSectionTmpl は埋め込んだ雛形を解析したもの。名前付きテンプレートを個別に実行する。
var statusSectionTmpl = template.Must(template.New("status_section").Parse(statusSectionTmplSrc))

// statusDisplayOrder は状況テーブルで status を並べる順。着手対象を上に置く。
var statusDisplayOrder = []Status{
	StatusInProgress, StatusAccepted, StatusDraft, StatusDone, StatusSuperseded, StatusDropped,
}

// renderLabel は名前付きテンプレートを data で実行した文字列を返す。
// テンプレートは埋め込みでゴールデンテストが担保するため、実行時に失敗し得ない。
func renderLabel(name string, data any) string {
	var b strings.Builder
	if err := statusSectionTmpl.ExecuteTemplate(&b, name, data); err != nil {
		panic(err)
	}

	return b.String()
}

// RenderStatusSection は README に埋め込む未完了ドキュメントの一覧を Markdown で返す。
// docs は表示したい順に並んでいる前提。載せるのはアクションが要る status、すなわち Status.IsOpen が
// true のものに限る。status 別の件数テーブルは出さない。done が増えるたびに変わり、ブランチ間で
// 頻繁に衝突するため。
func RenderStatusSection(docs []*Document) string {
	var b strings.Builder
	b.WriteString(renderLabel("header", nil))
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
		b.WriteString(renderLabel("empty", nil))
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
		return renderLabel("notitle", nil)
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
		s += renderLabel("skipped", d.SkippedTasks)
	}

	return s
}
