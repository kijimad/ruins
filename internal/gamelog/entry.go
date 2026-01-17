package gamelog

// LogEntry は複数のフラグメントからなるログエントリ
type LogEntry struct {
	Fragments []LogFragment `json:"fragments"`
}

// Text はエントリ全体のテキストを結合して返す
func (e LogEntry) Text() string {
	var result string
	for _, fragment := range e.Fragments {
		result += fragment.Text
	}
	return result
}

// IsEmpty はエントリが空かどうかを判定
func (e LogEntry) IsEmpty() bool {
	return len(e.Fragments) == 0
}
