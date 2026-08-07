package gamelog

// Logger はマークアップ文字列から色付きログを組み立てて出力する。
// 表示テキストはすべて Markup で積む。色はマークアップのタグで表す
type Logger struct {
	fragments []LogFragment
	store     *SafeSlice
}

// New は指定されたストアでLoggerを作成する。
// 本番: New(GameLog) など、グローバルストアを渡す。テスト: ローカルストアを渡す
func New(store *SafeSlice) *Logger {
	return &Logger{
		fragments: make([]LogFragment, 0),
		store:     store,
	}
}

// Log はログを出力する。ストアは初期化時に指定済み
func (l *Logger) Log() {
	l.appendToLog(l.store)
}

// appendToLog は積んだ断片を1エントリとしてストアへ追加し、バッファをクリアする
func (l *Logger) appendToLog(log *SafeSlice) {
	if len(l.fragments) == 0 {
		return
	}
	fragmentsCopy := make([]LogFragment, len(l.fragments))
	copy(fragmentsCopy, l.fragments)
	log.pushColoredEntry(LogEntry{Fragments: fragmentsCopy})
	l.fragments = l.fragments[:0]
}
