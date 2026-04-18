package config

import (
	"math/rand/v2"
	"os"
)

// Profile は設定プロファイルを表す
type Profile string

const (
	// ProfileProduction は本番環境プロファイル
	ProfileProduction Profile = "production"
	// ProfileDevelopment は開発環境プロファイル
	ProfileDevelopment Profile = "development"
)

// Config はアプリケーションの設定を管理する
type Config struct {
	// 設定のデフォルト値を決定するプロファイル
	Profile Profile `env:"RUINS_PROFILE" envDefault:"production"`

	// ゲームウィンドウの幅（ピクセル）
	WindowWidth int `env:"RUINS_WINDOW_WIDTH"`
	// ゲームウィンドウの高さ（ピクセル）
	WindowHeight int `env:"RUINS_WINDOW_HEIGHT"`
	// フルスクリーンで起動するかどうか
	Fullscreen bool `env:"RUINS_FULLSCREEN"`

	// デバッグモードを有効にする。有効時は設定情報をログ出力する
	Debug bool `env:"RUINS_DEBUG"`
	// ログレベルを指定する
	LogLevel string `env:"RUINS_LOG_LEVEL"`
	// 出力するログカテゴリをカンマ区切りで指定する
	LogCategories string `env:"RUINS_LOG_CATEGORIES"`
	// pprofサーバーを起動するかどうか
	DebugPProf bool `env:"RUINS_DEBUG_PPROF"`
	// pprofサーバーのポート番号
	PProfPort int `env:"RUINS_PPROF_PORT"`
	// パフォーマンスモニターを表示するかどうか
	ShowMonitor bool `env:"RUINS_SHOW_MONITOR"`
	// AI行動のデバッグ表示を有効にするかどうか
	ShowAIDebug bool `env:"RUINS_SHOW_AI_DEBUG"`
	// マップのデバッグ表示を有効にするかどうか
	ShowMapDebug bool `env:"RUINS_SHOW_MAP_DEBUG"`
	// エンカウントを無効化するかどうか
	NoEncounter bool `env:"RUINS_NO_ENCOUNTER"`

	// キャラクター作成をスキップして拠点から開始するかどうか
	QuickStart bool `env:"RUINS_QUICK_START"`
	// アニメーション演出を無効化するかどうか
	DisableAnimation bool `env:"RUINS_DISABLE_ANIMATION"`

	// 乱数シード。環境変数で指定すると再現可能になる。未指定の場合は自動生成される
	Seed uint64 `env:"RUINS_SEED"`
	// 乱数生成器。Seedから生成される
	RNG *rand.Rand

	// 描画のターゲットFPS
	TargetFPS int `env:"RUINS_TARGET_FPS"`
	// メモリプロファイルを取得するかどうか
	ProfileMemory bool `env:"RUINS_PROFILE_MEMORY"`
	// CPUプロファイルを取得するかどうか
	ProfileCPU bool `env:"RUINS_PROFILE_CPU"`
	// Mutexプロファイルを取得するかどうか
	ProfileMutex bool `env:"RUINS_PROFILE_MUTEX"`
	// トレースプロファイルを取得するかどうか
	ProfileTrace bool `env:"RUINS_PROFILE_TRACE"`
	// プロファイル出力先のディレクトリパス
	ProfilePath string `env:"RUINS_PROFILE_PATH"`
}

// ApplyProfileDefaults はプロファイルに基づいてデフォルト値を設定する
func (c *Config) ApplyProfileDefaults() {
	switch c.Profile {
	case ProfileProduction:
		c.applyProductionDefaults()
	case ProfileDevelopment:
		c.applyDevelopmentDefaults()
	default:
		// デフォルトは本番設定
		c.applyProductionDefaults()
	}
}

// applyProductionDefaults は本番環境のデフォルト値を設定
func (c *Config) applyProductionDefaults() {
	// ウィンドウ設定
	if os.Getenv("RUINS_WINDOW_WIDTH") == "" {
		c.WindowWidth = 960
	}
	if os.Getenv("RUINS_WINDOW_HEIGHT") == "" {
		c.WindowHeight = 720
	}
	if os.Getenv("RUINS_FULLSCREEN") == "" {
		c.Fullscreen = false
	}

	// デバッグ設定
	if os.Getenv("RUINS_DEBUG") == "" {
		c.Debug = false
	}
	if os.Getenv("RUINS_LOG_LEVEL") == "" {
		c.LogLevel = "info"
	}
	if os.Getenv("RUINS_LOG_CATEGORIES") == "" {
		c.LogCategories = ""
	}
	if os.Getenv("RUINS_DEBUG_PPROF") == "" {
		c.DebugPProf = false
	}
	if os.Getenv("RUINS_PPROF_PORT") == "" {
		c.PProfPort = 6060
	}
	if os.Getenv("RUINS_SHOW_MONITOR") == "" {
		c.ShowMonitor = false
	}
	if os.Getenv("RUINS_SHOW_MAP_DEBUG") == "" {
		c.ShowMapDebug = false
	}
	if os.Getenv("RUINS_NO_ENCOUNTER") == "" {
		c.NoEncounter = false
	}

	// ゲーム設定
	if os.Getenv("RUINS_QUICK_START") == "" {
		c.QuickStart = false
	}
	if os.Getenv("RUINS_DISABLE_ANIMATION") == "" {
		c.DisableAnimation = false
	}

	// パフォーマンス設定
	if os.Getenv("RUINS_TARGET_FPS") == "" {
		c.TargetFPS = 60
	}

	// プロファイル設定
	if os.Getenv("RUINS_PROFILE_MEMORY") == "" {
		c.ProfileMemory = false
	}
	if os.Getenv("RUINS_PROFILE_CPU") == "" {
		c.ProfileCPU = false
	}
	if os.Getenv("RUINS_PROFILE_MUTEX") == "" {
		c.ProfileMutex = false
	}
	if os.Getenv("RUINS_PROFILE_TRACE") == "" {
		c.ProfileTrace = false
	}
	if os.Getenv("RUINS_PROFILE_PATH") == "" {
		c.ProfilePath = "."
	}
}

// applyDevelopmentDefaults は開発環境のデフォルト値を設定
func (c *Config) applyDevelopmentDefaults() {
	if os.Getenv("RUINS_WINDOW_WIDTH") == "" {
		c.WindowWidth = 960
	}
	if os.Getenv("RUINS_WINDOW_HEIGHT") == "" {
		c.WindowHeight = 720
	}
	if os.Getenv("RUINS_FULLSCREEN") == "" {
		c.Fullscreen = false
	}

	// デバッグ設定
	if os.Getenv("RUINS_DEBUG") == "" {
		c.Debug = true
	}
	if os.Getenv("RUINS_LOG_LEVEL") == "" {
		c.LogLevel = "info"
	}
	if os.Getenv("RUINS_LOG_CATEGORIES") == "" {
		c.LogCategories = ""
	}
	if os.Getenv("RUINS_DEBUG_PPROF") == "" {
		c.DebugPProf = true
	}
	if os.Getenv("RUINS_PPROF_PORT") == "" {
		c.PProfPort = 6060
	}
	if os.Getenv("RUINS_SHOW_MONITOR") == "" {
		c.ShowMonitor = false
	}
	if os.Getenv("RUINS_SHOW_MAP_DEBUG") == "" {
		c.ShowMapDebug = false
	}
	if os.Getenv("RUINS_NO_ENCOUNTER") == "" {
		c.NoEncounter = false
	}

	// ゲーム設定
	if os.Getenv("RUINS_QUICK_START") == "" {
		c.QuickStart = true
	}
	if os.Getenv("RUINS_DISABLE_ANIMATION") == "" {
		c.DisableAnimation = false
	}

	// パフォーマンス設定
	if os.Getenv("RUINS_TARGET_FPS") == "" {
		c.TargetFPS = 60
	}

	// プロファイル設定
	if os.Getenv("RUINS_PROFILE_MEMORY") == "" {
		c.ProfileMemory = true
	}
	if os.Getenv("RUINS_PROFILE_CPU") == "" {
		c.ProfileCPU = false
	}
	if os.Getenv("RUINS_PROFILE_MUTEX") == "" {
		c.ProfileMutex = false
	}
	if os.Getenv("RUINS_PROFILE_TRACE") == "" {
		c.ProfileTrace = false
	}
	if os.Getenv("RUINS_PROFILE_PATH") == "" {
		c.ProfilePath = "./profiles" // 開発時は専用フォルダ
	}
}

// Validate は設定値の妥当性を検証する
func (c *Config) Validate() error {
	if c.WindowWidth < 320 {
		c.WindowWidth = 320
	}
	if c.WindowHeight < 240 {
		c.WindowHeight = 240
	}
	if c.TargetFPS < 1 {
		c.TargetFPS = 60
	}
	if c.PProfPort < 1024 || c.PProfPort > 65535 {
		c.PProfPort = 6060
	}

	return nil
}
