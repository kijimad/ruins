package logger

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"runtime"
	"time"
)

// Logger はカテゴリ付きロガー
type Logger struct {
	category Category
	fields   map[string]any
}

// New は新しいロガーを作成する
func New(category Category) *Logger {
	return &Logger{
		category: category,
		fields:   make(map[string]any),
	}
}

// WithField はフィールドを追加した新しいロガーを返す
func (l *Logger) WithField(key string, value any) *Logger {
	newLogger := &Logger{
		category: l.category,
		fields:   make(map[string]any),
	}
	maps.Copy(newLogger.fields, l.fields)
	newLogger.fields[key] = value
	return newLogger
}

// WithFields は複数フィールドを追加した新しいロガーを返す
func (l *Logger) WithFields(fields map[string]any) *Logger {
	newLogger := &Logger{
		category: l.category,
		fields:   make(map[string]any),
	}
	maps.Copy(newLogger.fields, l.fields)
	maps.Copy(newLogger.fields, fields)
	return newLogger
}

// Debug はデバッグレベルのログを出力する
func (l *Logger) Debug(msg string, keysAndValues ...any) {
	l.log(LevelDebug, msg, keysAndValues...)
}

// Info は情報レベルのログを出力する
func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.log(LevelInfo, msg, keysAndValues...)
}

// Warn は警告レベルのログを出力する
func (l *Logger) Warn(msg string, keysAndValues ...any) {
	l.log(LevelWarn, msg, keysAndValues...)
}

// Error はエラーレベルのログを出力する
func (l *Logger) Error(msg string, keysAndValues ...any) {
	l.log(LevelError, msg, keysAndValues...)
}

// Fatal は致命的エラーレベルのログを出力してプログラムを終了する
func (l *Logger) Fatal(msg string, keysAndValues ...any) {
	l.log(LevelFatal, msg, keysAndValues...)
	os.Exit(1)
}

// IsDebugEnabled はデバッグレベルが有効かチェックする
func (l *Logger) IsDebugEnabled() bool {
	categoryLevel, exists := globalConfig.CategoryLevels[l.category]
	if !exists {
		categoryLevel = globalConfig.DefaultLevel
	}
	return LevelDebug >= categoryLevel
}

// log は実際のログ出力処理を行う
func (l *Logger) log(level Level, msg string, keysAndValues ...any) {
	// カテゴリ別レベルチェック
	categoryLevel, exists := globalConfig.CategoryLevels[l.category]
	if !exists {
		categoryLevel = globalConfig.DefaultLevel
	}

	// レベルが不足していれば早期リターン
	if level < categoryLevel {
		return
	}

	// ログエントリを構築
	entry := make(map[string]any)
	entry["timestamp"] = time.Now().Format(TimeFormat)
	entry["level"] = level.String()
	entry["category"] = string(l.category)
	entry["message"] = msg

	// 呼び出し元情報を追加
	if pc, file, line, ok := runtime.Caller(2); ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			entry["caller"] = fmt.Sprintf("%s:%d", file, line)
			entry["function"] = fn.Name()
		}
	}

	// 固定フィールドを追加
	maps.Copy(entry, l.fields)

	// キー値ペアを追加
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if ok {
				entry[key] = keysAndValues[i+1]
			}
		}
	}

	// JSON形式で出力
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(entry); err != nil {
		fmt.Fprintf(os.Stderr, "ログ出力エラー: %v\n", err)
	}
}
