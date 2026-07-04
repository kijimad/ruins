package gamelog

import "strings"

// LogEntry は複数のフラグメントからなるログエントリ
type LogEntry struct {
	Fragments []LogFragment `json:"fragments"`
}

// Text はエントリ全体のテキストを結合して返す
func (e LogEntry) Text() string {
	var result strings.Builder
	for _, fragment := range e.Fragments {
		result.WriteString(fragment.Text)
	}
	return result.String()
}

// IsEmpty はエントリが空かどうかを判定
func (e LogEntry) IsEmpty() bool {
	return len(e.Fragments) == 0
}
