package gamelog

import (
	"sync"

	"github.com/kijimaD/ruins/internal/consts"
)

const (
	// DefaultMaxLogSize はデフォルトの最大ログサイズ
	DefaultMaxLogSize = 1000
	// FieldLogMaxSize はフィールドログの最大サイズ
	FieldLogMaxSize = 100
	// SceneLogMaxSize はシーンログの最大サイズ
	SceneLogMaxSize = 200
)

// SafeSlice はスレッドセーフなログストレージ
type SafeSlice struct {
	coloredEntries []LogEntry
	maxSize        int
	mu             sync.Mutex
}

// NewSafeSlice は指定されたサイズの新しいSafeSliceを作成する
func NewSafeSlice(maxSize int) *SafeSlice {
	if maxSize <= 0 {
		maxSize = DefaultMaxLogSize
	}
	return &SafeSlice{
		coloredEntries: make([]LogEntry, 0, maxSize),
		maxSize:        maxSize,
	}
}

// Push は新しいログを追加
func (s *SafeSlice) Push(message string) {
	// 単純な文字列を色付きログエントリに変換
	entry := LogEntry{
		Fragments: []LogFragment{{
			Text:  message,
			Color: consts.ColorWhite, // デフォルト色は白
		}},
	}
	s.pushColoredEntry(entry)
}

// GetRecent は最新N行を表示順で取得（文字列版）
// 色付きエントリから文字列に変換して返す
func (s *SafeSlice) GetRecent(lines int) []string {
	entries := s.GetRecentEntries(lines)
	result := make([]string, len(entries))
	for i, entry := range entries {
		result[i] = entry.Text() // LogEntryのTextメソッドで全フラグメントを結合
	}
	return result
}

// GetHistory は全履歴を表示順で取得（文字列版）
// 色付きエントリから文字列に変換して返す
func (s *SafeSlice) GetHistory() []string {
	entries := s.GetHistoryEntries()
	result := make([]string, len(entries))
	for i, entry := range entries {
		result[i] = entry.Text()
	}
	return result
}

// Clear は全ログを削除
func (s *SafeSlice) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.coloredEntries = []LogEntry{}
}

// Count は現在のログ行数
func (s *SafeSlice) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.coloredEntries)
}

// MaxHistory は履歴の最大保持行数
func (s *SafeSlice) MaxHistory() int {
	return s.maxSize
}

// pushColoredEntry は色付きエントリを内部的に追加する
func (s *SafeSlice) pushColoredEntry(entry LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 新しいエントリを追加
	s.coloredEntries = append(s.coloredEntries, entry)

	// 最大サイズを超えた場合、古いものから削除（FIFO）
	if len(s.coloredEntries) > s.maxSize {
		keepCount := s.maxSize
		if keepCount > len(s.coloredEntries) {
			keepCount = len(s.coloredEntries)
		}

		// 新しいスライスに最新の要素をコピー
		newEntries := make([]LogEntry, keepCount)
		copy(newEntries, s.coloredEntries[len(s.coloredEntries)-keepCount:])
		s.coloredEntries = newEntries
	}
}

// GetRecentEntries は最新のN行の色付きエントリを表示順で取得
func (s *SafeSlice) GetRecentEntries(lines int) []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 負の値や0の場合は空のスライスを返す
	if lines <= 0 || len(s.coloredEntries) == 0 {
		return []LogEntry{}
	}

	// 要求された行数とコンテンツの長さの小さい方を使用
	actualLines := lines
	if actualLines > len(s.coloredEntries) {
		actualLines = len(s.coloredEntries)
	}

	// 最新のactualLines行を取得（表示順）
	startIndex := len(s.coloredEntries) - actualLines
	result := make([]LogEntry, actualLines)
	copy(result, s.coloredEntries[startIndex:])

	return result
}

// GetHistoryEntries は全色付きエントリ履歴を表示順で取得
func (s *SafeSlice) GetHistoryEntries() []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]LogEntry, len(s.coloredEntries))
	copy(result, s.coloredEntries)

	return result
}
