package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
}
