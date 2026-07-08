package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyProfileDefaults_Production(t *testing.T) {
	t.Parallel()

	cfg := &Config{Profile: ProfileProduction}
	cfg.ApplyProfileDefaults()

	// 環境変数が未設定の場合にデフォルト値が適用される
	// テスト実行時に一部の環境変数が設定されている可能性があるため、
	// 環境変数に依存しないフィールドのみ検証する
	assert.Equal(t, 960, cfg.User.WindowWidth)
	assert.Equal(t, 720, cfg.User.WindowHeight)
	assert.False(t, cfg.SkipOpening)
	assert.Equal(t, 60, cfg.TargetFPS)
}

func TestApplyProfileDefaults_Development(t *testing.T) {
	t.Parallel()

	cfg := &Config{Profile: ProfileDevelopment}
	cfg.ApplyProfileDefaults()

	assert.Equal(t, 960, cfg.User.WindowWidth)
	assert.Equal(t, 720, cfg.User.WindowHeight)
	assert.True(t, cfg.SkipOpening)
}

func TestApplyProfileDefaults_UnknownProfile(t *testing.T) {
	t.Parallel()

	cfg := &Config{Profile: Profile("unknown")}
	cfg.ApplyProfileDefaults()

	// 不明なプロファイルは本番設定にフォールバックする
	assert.False(t, cfg.Debug)
	assert.False(t, cfg.SkipOpening)
}

func TestLoad(t *testing.T) {
	t.Parallel()

	cfg, err := Load()
	require.NoError(t, err)

	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.RNG)
	assert.Greater(t, cfg.User.WindowWidth, 0)
	assert.Greater(t, cfg.User.WindowHeight, 0)
	assert.Greater(t, cfg.TargetFPS, 0)
}
func TestValidate(t *testing.T) {
	t.Parallel()

	valid := func() *Config {
		return &Config{
			User:      UserConfig{WindowWidth: 1920, WindowHeight: 1080},
			TargetFPS: 144,
			PProfPort: 8080,
		}
	}

	t.Run("有効な値はエラーを返さない", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, valid().Validate())
	})

	t.Run("不正な値はエラーを返す", func(t *testing.T) {
		t.Parallel()

		cases := map[string]func(*Config){
			"ウィンドウ幅が最小値未満":  func(c *Config) { c.User.WindowWidth = 100 },
			"ウィンドウ高さが最小値未満": func(c *Config) { c.User.WindowHeight = 50 },
			"目標FPSが1未満":     func(c *Config) { c.TargetFPS = 0 },
			"pprofポートが下限未満": func(c *Config) { c.PProfPort = 80 },
			"pprofポートが上限超過": func(c *Config) { c.PProfPort = 70000 },
		}
		for name, mutate := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				cfg := valid()
				mutate(cfg)
				assert.Error(t, cfg.Validate())
			})
		}
	})
}
