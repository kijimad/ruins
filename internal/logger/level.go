package logger

// Level はログレベルを表す型
type Level int

const (
	// LevelDebug はデバッグレベルのログ
	LevelDebug Level = iota
	// LevelInfo は情報レベルのログ
	LevelInfo
	// LevelWarn は警告レベルのログ
	LevelWarn
	// LevelError はエラーレベルのログ
	LevelError
	// LevelFatal は致命的エラーレベルのログ
	LevelFatal
	// LevelIgnore はすべてのログを無視する（テスト用）
	LevelIgnore
)

// String はレベルを文字列で返す
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	case LevelIgnore:
		return "IGNORE"
	default:
		return "UNKNOWN"
	}
}

// parseLevel は文字列(環境変数由来)からレベルを解析する
func parseLevel(s string) Level {
	switch s {
	case "debug", "DEBUG":
		return LevelDebug
	case "info", "INFO":
		return LevelInfo
	case "warn", "WARN":
		return LevelWarn
	case "error", "ERROR":
		return LevelError
	case "fatal", "FATAL":
		return LevelFatal
	case "ignore", "IGNORE":
		return LevelIgnore
	default:
		return LevelInfo
	}
}
