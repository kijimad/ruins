package designdoc

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// DefaultDir は設計ドキュメントが置かれる既定のディレクトリ。
const DefaultDir = "docs/design"

// templateFile は雛形。実ドキュメントではないので読み込み対象から除外する。
const templateFile = "tmpl.md"

// LoadDir は dir 直下の設計ドキュメントを解析して返す。tmpl.md と .md 以外は除外し、ファイル名昇順で並べる。
func LoadDir(dir string) ([]*Document, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("%s の読み込みに失敗: %w", dir, err)
	}

	var docs []*Document
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".md" || name == templateFile {
			continue
		}

		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s の読み込みに失敗: %w", path, err)
		}

		doc, err := Parse(path, string(content))
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}

	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })

	return docs, nil
}

// Severity は検証結果の深刻度を表す。
type Severity int

const (
	// SeverityError は構造上の不正。CI を落とす。
	SeverityError Severity = iota
	// SeverityWarn は表記ゆれや整合の崩れ。表示するが CI は落とさない。
	SeverityWarn
)

// String は深刻度のラベルを返す。
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "ERROR"
	case SeverityWarn:
		return "WARN"
	}

	return "UNKNOWN"
}

// Problem は検証で見つかった1件の問題を表す。
type Problem struct {
	Path     string
	Severity Severity
	Message  string
}

// Validate はドキュメント群の frontmatter を検証して問題の一覧を返す。
// 構造の不正は Error、表記ゆれや status と進捗の食い違いは Warn とする。
func Validate(docs []*Document) []Problem {
	var problems []Problem
	add := func(path string, sev Severity, msg string) {
		problems = append(problems, Problem{Path: path, Severity: sev, Message: msg})
	}

	for _, doc := range docs {
		if !doc.HasFront {
			add(doc.Path, SeverityError, "frontmatter がない。backfill で付与する")
			continue
		}
		if !doc.Front.Status.Valid() {
			add(doc.Path, SeverityError, fmt.Sprintf("status が不正: %q", doc.Front.Status))
		}
		if !doc.Front.Auto.Valid() {
			add(doc.Path, SeverityError, fmt.Sprintf("auto が不正: %q", doc.Front.Auto))
		}
		for _, tag := range doc.Front.Tags {
			if !slices.Contains(KnownTags, tag) {
				add(doc.Path, SeverityWarn, fmt.Sprintf("未知のタグ %q。KnownTags を確認する", tag))
			}
		}
		// done は「open な `- [ ]` がゼロ」を満たす不変条件。裏付けのない done を弾く。
		// 着手しないと決めたタスクは `- [~]` にすれば open から外れ、done にできる。
		if doc.Front.Status == StatusDone && doc.OpenTasks > 0 {
			add(doc.Path, SeverityError, fmt.Sprintf("status=done だが未チェックのタスクが %d 件ある。完了するか `- [~]` にする", doc.OpenTasks))
		}
		if doc.Front.Status == StatusInProgress && !doc.HasProgress {
			add(doc.Path, SeverityWarn, "status=in-progress だが進捗セクションがない")
		}
	}

	return problems
}

// HasError は問題一覧に Error が含まれるかを返す。
func HasError(problems []Problem) bool {
	for _, p := range problems {
		if p.Severity == SeverityError {
			return true
		}
	}

	return false
}

// BackfillDir は dir 直下の frontmatter を欠くドキュメントに既定値を付与し、変更したファイルのパスを返す。
// 冪等なので、既に付与済みのファイルには触れない。
func BackfillDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("%s の読み込みに失敗: %w", dir, err)
	}

	var changed []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".md" || name == templateFile {
			continue
		}

		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s の読み込みに失敗: %w", path, err)
		}

		result, ok, err := Backfill(string(content))
		if err != nil {
			return nil, fmt.Errorf("%s の backfill に失敗: %w", path, err)
		}
		if !ok {
			continue
		}

		if err := os.WriteFile(path, []byte(result), 0644); err != nil {
			return nil, fmt.Errorf("%s の書き込みに失敗: %w", path, err)
		}
		changed = append(changed, path)
	}

	sort.Strings(changed)

	return changed, nil
}

// Backfill は frontmatter を欠くドキュメントに決定的な既定値を付与した内容を返す。
// 既に frontmatter があれば内容を変えず changed=false を返す。冪等なので何度でも流せる。
func Backfill(content string) (result string, changed bool, err error) {
	doc, err := Parse("", content)
	if err != nil {
		return "", false, err
	}
	if doc.HasFront {
		return content, false, nil
	}

	front := Frontmatter{
		Status: InferStatus(doc),
		Tags:   []string{},
		Auto:   AutoNeedsDecision,
	}

	rendered, err := Render(front, strings.TrimPrefix(content, "\n"))
	if err != nil {
		return "", false, err
	}

	return rendered, true, nil
}
