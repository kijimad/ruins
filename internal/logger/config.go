package logger

import (
	"strings"
)

const (
	// TimeFormat はタイムスタンプ形式
	TimeFormat = "2006-01-02T15:04:05.000Z"
)

// Config はログ設定
type Config struct {
	// デフォルトログレベル
	DefaultLevel Level

	// カテゴリ別ログレベル
	CategoryLevels map[Category]Level
}

// グローバル設定（初期化時は空の設定）
// Config構造体のデフォルト値、環境変数によって上書きされるので、ここでの初期値はプロダクトコードでは尊重されない
// 環境変数を読み込まないテスト時に使われる
var globalConfig = Config{
	DefaultLevel:   LevelIgnore,
	CategoryLevels: make(map[Category]Level),
}

// parseCategoryLevels はカテゴリ別レベル設定を解析する
func parseCategoryLevels(s string) map[Category]Level {
	result := make(map[Category]Level)
	pairs := strings.Split(s, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), "=")
		if len(parts) == 2 {
			category := Category(parts[0])
			level := parseLevel(parts[1])
			result[category] = level
		}
	}
	return result
}

// LoadFromConfig はconfigパッケージの設定を読み込む
func LoadFromConfig(logLevel, logCategories string) {
	config := Config{
		DefaultLevel:   parseLevel(logLevel),
		CategoryLevels: make(map[Category]Level),
	}

	// configパッケージからのカテゴリ別ログレベル設定
	if logCategories != "" {
		config.CategoryLevels = parseCategoryLevels(logCategories)
	}

	globalConfig = config
}

// SetConfig はグローバル設定を更新する（テスト用）
func SetConfig(config Config) {
	globalConfig = config
}

// ResetConfig は設定をリセットする（テスト用）
func ResetConfig() {
	globalConfig = Config{
		DefaultLevel:   LevelIgnore,
		CategoryLevels: make(map[Category]Level),
	}
}
