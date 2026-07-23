package designdoc

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// fence は frontmatter を囲むデリミタ。
const fence = "---"

var (
	reTitle           = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	reDocNumber       = regexp.MustCompile(`_(\d+)\.md$`)
	reProgressHeading = regexp.MustCompile(`(?m)^##\s+進捗\s*$`)
	reAnyHeading      = regexp.MustCompile(`(?m)^##\s`)
	reOpenTask        = regexp.MustCompile(`(?m)^- \[ \]`)
	reDoneTask        = regexp.MustCompile(`(?m)^- \[x\]`)
	// reSkipTask は「意図的に着手しない」タスク。不採用・見送りを表す。open にも done にも数えない。
	reSkipTask = regexp.MustCompile(`(?m)^- \[~\]`)
)

// Parse は設計ドキュメントの内容を解析する。path は診断メッセージに使うだけで読み込みはしない。
func Parse(path string, content string) (*Document, error) {
	doc := &Document{Path: path}

	front, body, hasFront, err := splitFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("%s の frontmatter 解析に失敗: %w", path, err)
	}
	doc.HasFront = hasFront
	doc.Front = front
	doc.Body = body

	if m := reTitle.FindStringSubmatch(body); m != nil {
		doc.Title = m[1]
	}
	if m := reDocNumber.FindStringSubmatch(path); m != nil {
		// 抽出済みの数字部分なので Atoi は失敗しない。念のため失敗時は 0 のままにする。
		if n, convErr := strconv.Atoi(m[1]); convErr == nil {
			doc.Number = n
		}
	}
	countProgress(doc)

	return doc, nil
}

// splitFrontmatter は先頭の YAML frontmatter と本文を分離する。frontmatter が無ければ hasFront が false になる。
func splitFrontmatter(content string) (Frontmatter, string, bool, error) {
	if !strings.HasPrefix(content, fence+"\n") {
		return Frontmatter{}, content, false, nil
	}

	rest := content[len(fence)+1:]
	end := strings.Index(rest, "\n"+fence)
	if end < 0 {
		return Frontmatter{}, content, false, fmt.Errorf("閉じデリミタ %q が見つからない", fence)
	}

	yamlPart := rest[:end]
	// 閉じデリミタの行を飛ばし、続く空行も落として本文が見出しから始まるようにする。
	// Render は frontmatter と本文の間に必ず空行を1つ入れるので、往復しても安定する。
	body := strings.TrimLeft(rest[end+1+len(fence):], "\n")

	var front Frontmatter
	if err := yaml.Unmarshal([]byte(yamlPart), &front); err != nil {
		return Frontmatter{}, content, false, fmt.Errorf("YAML の unmarshal に失敗: %w", err)
	}

	return front, body, true, nil
}

// countProgress は本文の `## 進捗` セクションからチェックボックス数を数える。
func countProgress(doc *Document) {
	loc := reProgressHeading.FindStringIndex(doc.Body)
	if loc == nil {
		return
	}
	doc.HasProgress = true

	section := doc.Body[loc[1]:]
	// 次の `## ` 見出しがあればそこまでを進捗セクションとする。
	if next := reAnyHeading.FindStringIndex(section); next != nil {
		section = section[:next[0]]
	}

	doc.OpenTasks = len(reOpenTask.FindAllString(section, -1))
	doc.DoneTasks = len(reDoneTask.FindAllString(section, -1))
	doc.SkippedTasks = len(reSkipTask.FindAllString(section, -1))
}

// InferStatus は進捗チェックボックスから status を決定的に導出する。
// LLM 推論を挟まないため、同じ入力に対して常に同じ結果を返す。
//
//   - 進捗セクションが無い、または項目0件 → draft
//   - 未完タスクが残る → in-progress
//   - 全て完了 → done
//
// superseded・dropped・accepted は進捗からは判定できないため、人が個別に設定する。
func InferStatus(doc *Document) Status {
	switch {
	case !doc.HasProgress:
		return StatusDraft
	case doc.OpenTasks > 0:
		return StatusInProgress
	case doc.DoneTasks > 0:
		return StatusDone
	default:
		return StatusDraft
	}
}

// Render は frontmatter と本文から、ファイルに書き戻す文字列を組み立てる。
func Render(front Frontmatter, body string) (string, error) {
	yamlPart, err := yaml.Marshal(front)
	if err != nil {
		return "", fmt.Errorf("frontmatter の marshal に失敗: %w", err)
	}

	var b strings.Builder
	b.WriteString(fence + "\n")
	// strings.Builder.Write はエラーを返さない。
	_, _ = b.Write(yamlPart)
	b.WriteString(fence + "\n\n")
	b.WriteString(body)

	return b.String(), nil
}
