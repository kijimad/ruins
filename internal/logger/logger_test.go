package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// captureOutput はos.Stdoutの出力をキャプチャする
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestLoggerNew(t *testing.T) {
	t.Parallel()
	logger := New(CategoryDebug)
	if logger.category != CategoryDebug {
		t.Errorf("期待値: %s, 実際: %s", CategoryDebug, logger.category)
	}
	if len(logger.fields) != 0 {
		t.Errorf("fieldsは空であるべき")
	}
}

func TestLoggerWithField(t *testing.T) {
	t.Parallel()
	logger := New(CategoryDebug)
	newLogger := logger.WithField("key", "value")

	if len(logger.fields) != 0 {
		t.Errorf("元のロガーは変更されないべき")
	}
	if newLogger.fields["key"] != "value" {
		t.Errorf("フィールドが追加されていない")
	}
}

func TestLoggerWithFields(t *testing.T) {
	t.Parallel()
	logger := New(CategoryDebug)
	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	newLogger := logger.WithFields(fields)

	if len(logger.fields) != 0 {
		t.Errorf("元のロガーは変更されないべき")
	}
	if newLogger.fields["key1"] != "value1" || newLogger.fields["key2"] != 42 {
		t.Errorf("フィールドが正しく追加されていない")
	}
}

//nolint:paralleltest // modifies global config
func TestLogLevelFiltering(t *testing.T) {
	// テスト用の設定
	SetConfig(Config{
		DefaultLevel:   LevelInfo,
		CategoryLevels: make(map[Category]Level),
	})
	defer ResetConfig()

	logger := New(CategoryDebug)

	// Debugログは出力されない
	output := captureOutput(func() {
		logger.Debug("デバッグメッセージ")
	})
	if output != "" {
		t.Errorf("DEBUGレベルのログは出力されないべき")
	}

	// Infoログは出力される
	output = captureOutput(func() {
		logger.Info("情報メッセージ")
	})
	if output == "" {
		t.Errorf("INFOレベルのログは出力されるべき")
	}
}

//nolint:paralleltest // modifies global config
func TestContextLevelFiltering(t *testing.T) {
	// カテゴリ別設定
	SetConfig(Config{
		DefaultLevel: LevelWarn,
		CategoryLevels: map[Category]Level{
			CategoryDebug: LevelDebug,
		},
	})
	defer ResetConfig()

	// Battleカテゴリはデバッグレベルが有効
	battleLogger := New(CategoryDebug)
	output := captureOutput(func() {
		battleLogger.Debug("戦闘デバッグ")
	})
	if output == "" {
		t.Errorf("Battleカテゴリのデバッグログは出力されるべき")
	}

	// Moveカテゴリはデフォルト（Warn）レベル
	moveLogger := New(CategoryMove)
	output = captureOutput(func() {
		moveLogger.Info("移動情報")
	})
	if output != "" {
		t.Errorf("Moveカテゴリの情報ログは出力されないべき")
	}
}

//nolint:paralleltest // modifies global config
func TestJSONOutput(t *testing.T) {
	SetConfig(Config{
		DefaultLevel:   LevelDebug,
		CategoryLevels: make(map[Category]Level),
	})
	defer ResetConfig()

	logger := New(CategoryDebug)
	output := captureOutput(func() {
		logger.Info("テストメッセージ", "key1", "value1", "key2", 42)
	})

	// JSON解析
	var entry map[string]interface{}
	err := json.Unmarshal([]byte(output), &entry)
	require.NoError(t, err, "JSON解析エラー")

	// 必須フィールドの確認
	const expectedLevel = "INFO"
	if entry["level"] != expectedLevel {
		t.Errorf("levelが正しくない: %v", entry["level"])
	}
	if entry["category"] != "debug" {
		t.Errorf("categoryが正しくない: %v", entry["category"])
	}
	if entry["message"] != "テストメッセージ" {
		t.Errorf("messageが正しくない: %v", entry["message"])
	}
	if entry["key1"] != "value1" {
		t.Errorf("key1が正しくない: %v", entry["key1"])
	}
	if entry["key2"] != float64(42) { // JSONでは数値はfloat64になる
		t.Errorf("key2が正しくない: %v", entry["key2"])
	}
}

//nolint:paralleltest // modifies global config
func TestIsDebugEnabled(t *testing.T) {
	// t.Parallel() disabled: modifies global config
	tests := []struct {
		name          string
		config        Config
		category      Category
		expectEnabled bool
	}{
		{
			name: "デフォルトレベルがDebug",
			config: Config{
				DefaultLevel:   LevelDebug,
				CategoryLevels: make(map[Category]Level),
			},
			category:      CategoryDebug,
			expectEnabled: true,
		},
		{
			name: "デフォルトレベルがInfo",
			config: Config{
				DefaultLevel:   LevelInfo,
				CategoryLevels: make(map[Category]Level),
			},
			category:      CategoryDebug,
			expectEnabled: false,
		},
		{
			name: "カテゴリ別設定でDebug有効",
			config: Config{
				DefaultLevel: LevelInfo,
				CategoryLevels: map[Category]Level{
					CategoryDebug: LevelDebug,
				},
			},
			category:      CategoryDebug,
			expectEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel() disabled: modifies global config
			SetConfig(tt.config)
			defer ResetConfig()

			logger := New(tt.category)
			if logger.IsDebugEnabled() != tt.expectEnabled {
				t.Errorf("期待値: %v, 実際: %v", tt.expectEnabled, logger.IsDebugEnabled())
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	t.Parallel() // parseLevel is a pure function, safe for parallel execution
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"fatal", LevelFatal},
		{"FATAL", LevelFatal},
		{"ignore", LevelIgnore},
		{"IGNORE", LevelIgnore},
		{"unknown", LevelInfo}, // デフォルト
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, 期待値: %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseCategoryLevels(t *testing.T) {
	t.Parallel() // parseCategoryLevels is a pure function, safe for parallel execution
	input := "battle=debug,render=warn,invalid"
	result := parseCategoryLevels(input)

	if result[CategoryDebug] != LevelDebug {
		t.Errorf("battleカテゴリのレベルが正しくない")
	}
	if result[CategoryRender] != LevelWarn {
		t.Errorf("renderカテゴリのレベルが正しくない")
	}
	if _, exists := result["invalid"]; exists {
		t.Errorf("無効な形式は無視されるべき")
	}
}

//nolint:paralleltest // modifies global config
func TestLoggerOutput(t *testing.T) {
	// t.Parallel() disabled: modifies global config
	SetConfig(Config{
		DefaultLevel:   LevelDebug,
		CategoryLevels: make(map[Category]Level),
	})
	t.Cleanup(ResetConfig)

	logger := New(CategoryDebug).WithField("session", "test123")

	// 各レベルのテスト
	tests := []struct {
		name     string
		logFunc  func(string, ...interface{})
		level    string
		contains []string
	}{
		{
			name:     "Debug",
			logFunc:  logger.Debug,
			level:    "DEBUG",
			contains: []string{"デバッグメッセージ", "DEBUG", "debug", "session", "test123"},
		},
		{
			name:     "Info",
			logFunc:  logger.Info,
			level:    "INFO",
			contains: []string{"情報メッセージ", "INFO", "debug"},
		},
		{
			name:     "Warn",
			logFunc:  logger.Warn,
			level:    "WARN",
			contains: []string{"警告メッセージ", "WARN", "debug"},
		},
		{
			name:     "Error",
			logFunc:  logger.Error,
			level:    "ERROR",
			contains: []string{"エラーメッセージ", "ERROR", "debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel() disabled: parent test modifies global config
			output := captureOutput(func() {
				tt.logFunc(tt.contains[0])
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("出力に %q が含まれていない: %s", expected, output)
				}
			}
		})
	}
}

//nolint:paralleltest
func TestIgnoreLevel(t *testing.T) {
	// ignoreレベルではすべてのログが出力されない
	SetConfig(Config{
		DefaultLevel:   LevelIgnore,
		CategoryLevels: make(map[Category]Level),
	})
	defer ResetConfig()

	logger := New(CategoryDebug)

	// すべてのレベルのログが出力されない
	levels := []struct {
		name string
		fn   func(string, ...interface{})
	}{
		{"Debug", logger.Debug},
		{"Info", logger.Info},
		{"Warn", logger.Warn},
		{"Error", logger.Error},
	}

	for _, level := range levels {
		t.Run(level.name, func(t *testing.T) {
			output := captureOutput(func() {
				level.fn("テストメッセージ")
			})
			if output != "" {
				t.Errorf("%sレベルのログは出力されないべき（ignoreレベル設定時）: %s", level.name, output)
			}
		})
	}
}
