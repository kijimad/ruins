package config

import (
	"fmt"
	"math/rand/v2"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/kijimaD/ruins/internal/logger"
)

// Load は環境変数から設定を読み込み、新しいConfigインスタンスを返す
// Seedが環境変数で指定されていない場合は、ランダム値を生成する
func Load() (*Config, error) {
	cfg := &Config{}

	// プロファイルを最初に決定(デフォルトはproduction)
	profile := os.Getenv("RUINS_PROFILE")
	if profile == "" {
		cfg.Profile = ProfileProduction
	} else {
		cfg.Profile = Profile(profile)
	}

	// プロファイルに基づくデフォルト値を設定
	cfg.ApplyProfileDefaults()

	// 永続化されたユーザー設定をファイルから読み込んで上書きする。
	// 失敗してもデフォルト値で起動を継続する。ファイルの生成は EnsureUserConfigFile が担う
	if err := cfg.loadUserConfig(); err != nil {
		logger.New(logger.CategoryLoad).Warn("ユーザー設定の読み込みに失敗しました。デフォルト値で継続します", "error", err)
	}

	// 環境変数で明示的に設定された値で上書きする。優先順位はデフォルト < ファイル < 環境変数
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("設定の読み込みに失敗しました: %w", err)
	}

	// Seedが未設定の場合はランダム値を生成
	if os.Getenv("RUINS_SEED") == "" {
		cfg.Seed = rand.Uint64()
	}
	// SeedからRNGを生成
	cfg.RNG = rand.New(rand.NewPCG(cfg.Seed, 0))

	cfg.Validate()

	return cfg, nil
}

// String は設定の文字列表現を返す（デバッグ用）
func (c *Config) String() string {
	return fmt.Sprintf(`Config{
	Profile: %s,
	WindowWidth: %d, WindowHeight: %d,
	Debug: %t, LogLevel: %s, LogCategories: %s, DebugPProf: %t, PProfPort: %d,
	Seed: %d,
	TargetFPS: %d,
	ProfileMemory: %t, ProfileCPU: %t, ProfileMutex: %t, ProfileTrace: %t,
	ProfilePath: %s
}`,
		c.Profile,
		c.User.WindowWidth, c.User.WindowHeight,
		c.Debug, c.LogLevel, c.LogCategories, c.DebugPProf, c.PProfPort,
		c.Seed,
		c.TargetFPS,
		c.ProfileMemory, c.ProfileCPU, c.ProfileMutex, c.ProfileTrace,
		c.ProfilePath)
}
