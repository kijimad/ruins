package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, CategoryDebug, logger.category)
	assert.Empty(t, logger.fields, "fieldsは空であるべき")
}

func TestLoggerWithField(t *testing.T) {
	t.Parallel()
	logger := New(CategoryDebug)
	newLogger := logger.WithField("key", "value")

	assert.Empty(t, logger.fields, "元のロガーは変更されないべき")
	assert.Equal(t, "value", newLogger.fields["key"], "フィールドが追加されていない")
}

func TestLoggerWithFields(t *testing.T) {
	t.Parallel()
	logger := New(CategoryDebug)
	fields := map[string]any{
		"key1": "value1",
		"key2": 42,
	}
	newLogger := logger.WithFields(fields)

	assert.Empty(t, logger.fields, "元のロガーは変更されないべき")
	assert.Equal(t, "value1", newLogger.fields["key1"], "フィールドが正しく追加されていない")
	assert.Equal(t, 42, newLogger.fields["key2"], "フィールドが正しく追加されていない")
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
	assert.Empty(t, output, "DEBUGレベルのログは出力されないべき")

	// Infoログは出力される
	output = captureOutput(func() {
		logger.Info("情報メッセージ")
	})
	assert.NotEmpty(t, output, "INFOレベルのログは出力されるべき")
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
	assert.NotEmpty(t, output, "Battleカテゴリのデバッグログは出力されるべき")

	// Moveカテゴリはデフォルト（Warn）レベル
	moveLogger := New(CategoryMove)
	output = captureOutput(func() {
		moveLogger.Info("移動情報")
	})
	assert.Empty(t, output, "Moveカテゴリの情報ログは出力されないべき")
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
	var entry map[string]any
	err := json.Unmarshal([]byte(output), &entry)
	require.NoError(t, err, "JSON解析エラー")

	// 必須フィールドの確認
	const expectedLevel = "INFO"
	assert.Equal(t, expectedLevel, entry["level"], "levelが正しくない")
	assert.Equal(t, "debug", entry["category"], "categoryが正しくない")
	assert.Equal(t, "テストメッセージ", entry["message"], "messageが正しくない")
	assert.Equal(t, "value1", entry["key1"], "key1が正しくない")
	assert.Equal(t, float64(42), entry["key2"], "key2が正しくない") // JSONでは数値はfloat64になる
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
			assert.Equal(t, tt.expectEnabled, logger.IsDebugEnabled())
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
			assert.Equal(t, tt.expected, result, "parseLevel(%q)", tt.input)
		})
	}
}

//nolint:paralleltest // modifies global config
func TestLoadFromConfig(t *testing.T) {
	defer ResetConfig()

	t.Run("デフォルトレベルのみ", func(t *testing.T) {
		LoadFromConfig("debug", "")
		require.Equal(t, LevelDebug, globalConfig.DefaultLevel)
		require.Empty(t, globalConfig.CategoryLevels)
	})

	t.Run("カテゴリ別設定あり", func(t *testing.T) {
		LoadFromConfig("info", "battle=debug,render=warn")
		require.Equal(t, LevelInfo, globalConfig.DefaultLevel)
		require.Equal(t, LevelDebug, globalConfig.CategoryLevels[CategoryDebug])
		require.Equal(t, LevelWarn, globalConfig.CategoryLevels[CategoryRender])
	})

	t.Run("不明なレベルはInfoになる", func(t *testing.T) {
		LoadFromConfig("unknown", "")
		require.Equal(t, LevelInfo, globalConfig.DefaultLevel)
	})
}

func TestLevelString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
		{LevelIgnore, "IGNORE"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, tt.level.String())
		})
	}
}

func TestParseCategoryLevels(t *testing.T) {
	t.Parallel() // parseCategoryLevels is a pure function, safe for parallel execution
	input := "battle=debug,render=warn,invalid"
	result := parseCategoryLevels(input)

	assert.Equal(t, LevelDebug, result[CategoryDebug], "battleカテゴリのレベルが正しくない")
	assert.Equal(t, LevelWarn, result[CategoryRender], "renderカテゴリのレベルが正しくない")
	_, exists := result["invalid"]
	assert.False(t, exists, "無効な形式は無視されるべき")
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
		logFunc  func(string, ...any)
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
				assert.Contains(t, output, expected)
			}
		})
	}
}

//nolint:paralleltest // グローバルなロガー設定を変更するため並列化しない
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
		fn   func(string, ...any)
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
			assert.Empty(t, output, "%sレベルのログは出力されないべき（ignoreレベル設定時）", level.name)
		})
	}
}
