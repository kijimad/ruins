package config

import (
	"fmt"
	"math/rand/v2"
	"os"

	"github.com/caarlos0/env/v11"
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

	// 環境変数で明示的に設定された値で上書き
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("設定の読み込みに失敗しました: %w", err)
	}

	// Seedが未設定の場合はランダム値を生成
	if os.Getenv("RUINS_SEED") == "" {
		cfg.Seed = rand.Uint64()
	}
	// SeedからRNGを生成
	cfg.RNG = rand.New(rand.NewPCG(cfg.Seed, 0))

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("設定の検証に失敗しました: %w", err)
	}

	return cfg, nil
}

// String は設定の文字列表現を返す（デバッグ用）
func (c *Config) String() string {
	return fmt.Sprintf(`Config{
	Profile: %s,
	WindowWidth: %d, WindowHeight: %d, Fullscreen: %t,
	Debug: %t, LogLevel: %s, LogCategories: %s, DebugPProf: %t, PProfPort: %d,
	StartingState: %s,
	Seed: %d,
	TargetFPS: %d,
	ProfileMemory: %t, ProfileCPU: %t, ProfileMutex: %t, ProfileTrace: %t,
	ProfilePath: %s
}`,
		c.Profile,
		c.WindowWidth, c.WindowHeight, c.Fullscreen,
		c.Debug, c.LogLevel, c.LogCategories, c.DebugPProf, c.PProfPort,
		c.StartingState,
		c.Seed,
		c.TargetFPS,
		c.ProfileMemory, c.ProfileCPU, c.ProfileMutex, c.ProfileTrace,
		c.ProfilePath)
}
