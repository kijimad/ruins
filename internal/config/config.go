package config

import (
	"os"

	"github.com/caarlos0/env/v11"
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
	// 環境プロファイル
	Profile Profile `env:"RUINS_PROFILE" envDefault:"production"`

	// ゲームウィンドウ設定
	WindowWidth  int  `env:"RUINS_WINDOW_WIDTH"`
	WindowHeight int  `env:"RUINS_WINDOW_HEIGHT"`
	Fullscreen   bool `env:"RUINS_FULLSCREEN"`

	// デバッグ設定
	Debug         bool   `env:"RUINS_DEBUG"`
	LogLevel      string `env:"RUINS_LOG_LEVEL"`
	LogCategories string `env:"RUINS_LOG_CATEGORIES"`
	DebugPProf    bool   `env:"RUINS_DEBUG_PPROF"`
	PProfPort     int    `env:"RUINS_PPROF_PORT"`
	ShowMonitor   bool   `env:"RUINS_SHOW_MONITOR"`
	ShowAIDebug   bool   `env:"RUINS_SHOW_AI_DEBUG"`
	ShowMapDebug  bool   `env:"RUINS_SHOW_MAP_DEBUG"`
	NoEncounter   bool   `env:"RUINS_NO_ENCOUNTER"`

	// ゲーム設定
	StartingState    string `env:"RUINS_STARTING_STATE"`
	DisableAnimation bool   `env:"RUINS_DISABLE_ANIMATION"`

	// テスト用固定シード。デバッグ時に指定して再現するのに使う
	TestSeed *uint64

	// パフォーマンス設定
	TargetFPS     int    `env:"RUINS_TARGET_FPS"`
	ProfileMemory bool   `env:"RUINS_PROFILE_MEMORY"`
	ProfileCPU    bool   `env:"RUINS_PROFILE_CPU"`
	ProfileMutex  bool   `env:"RUINS_PROFILE_MUTEX"`
	ProfileTrace  bool   `env:"RUINS_PROFILE_TRACE"`
	ProfilePath   string `env:"RUINS_PROFILE_PATH"`
}

// load は環境変数から設定を読み込む
func load() (*Config, error) {
	cfg := &Config{}

	// プロファイルを最初に決定(デフォルトはproduction)
	profile := os.Getenv("RUINS_PROFILE")
	if profile == "" {
		cfg.Profile = ProfileProduction
	} else {
		cfg.Profile = Profile(profile)
	}

	// プロファイルに基づくデフォルト値を設定
	cfg.applyProfileDefaults()

	// 環境変数で明示的に設定された値で上書き
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyProfileDefaults はプロファイルに基づいてデフォルト値を設定する
func (c *Config) applyProfileDefaults() {
	switch c.Profile {
	case ProfileDevelopment:
		c.applyDevelopmentDefaults()
	case ProfileProduction:
		c.applyProductionDefaults()
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
	if os.Getenv("RUINS_STARTING_STATE") == "" {
		c.StartingState = "main_menu"
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
	if os.Getenv("RUINS_STARTING_STATE") == "" {
		c.StartingState = "town"
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
