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
	assert.Equal(t, 960, cfg.WindowWidth)
	assert.Equal(t, 720, cfg.WindowHeight)
	assert.False(t, cfg.SkipOpening)
	assert.Equal(t, 60, cfg.TargetFPS)
}

func TestApplyProfileDefaults_Development(t *testing.T) {
	t.Parallel()

	cfg := &Config{Profile: ProfileDevelopment}
	cfg.ApplyProfileDefaults()

	assert.Equal(t, 960, cfg.WindowWidth)
	assert.Equal(t, 720, cfg.WindowHeight)
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
	assert.Greater(t, cfg.WindowWidth, 0)
	assert.Greater(t, cfg.WindowHeight, 0)
	assert.Greater(t, cfg.TargetFPS, 0)
}

func TestConfig_String(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Profile:      ProfileProduction,
		WindowWidth:  960,
		WindowHeight: 720,
		TargetFPS:    60,
		LogLevel:     "info",
		PProfPort:    6060,
		ProfilePath:  ".",
	}

	s := cfg.String()
	assert.Contains(t, s, "production")
	assert.Contains(t, s, "960")
	assert.Contains(t, s, "720")
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("無効な値が修正される", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			WindowWidth:  100, // 最小値以下
			WindowHeight: 50,  // 最小値以下
			TargetFPS:    0,   // 無効
			PProfPort:    80,  // 範囲外
		}

		err := cfg.Validate()
		assert.NoError(t, err)

		assert.Equal(t, 320, cfg.WindowWidth)
		assert.Equal(t, 240, cfg.WindowHeight)
		assert.Equal(t, 60, cfg.TargetFPS)
		assert.Equal(t, 6060, cfg.PProfPort)
	})

	t.Run("有効な値は変更されない", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			WindowWidth:  1920,
			WindowHeight: 1080,
			TargetFPS:    144,
			PProfPort:    8080,
		}

		err := cfg.Validate()
		assert.NoError(t, err)

		assert.Equal(t, 1920, cfg.WindowWidth)
		assert.Equal(t, 1080, cfg.WindowHeight)
		assert.Equal(t, 144, cfg.TargetFPS)
		assert.Equal(t, 8080, cfg.PProfPort)
	})

	t.Run("PProfPortが上限を超える場合", func(t *testing.T) {
		t.Parallel()

		cfg := &Config{
			WindowWidth:  960,
			WindowHeight: 720,
			TargetFPS:    60,
			PProfPort:    70000,
		}

		err := cfg.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 6060, cfg.PProfPort)
	})
}
