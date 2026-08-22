package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDangerLevel(t *testing.T) {
	t.Parallel()

	t.Run("危険度は1始まりで最小は1", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, dangerLevel(0))
		assert.Equal(t, 1, dangerLevel(dangerDaysPerLevel-1))
	})

	t.Run("1段ぶんの日数で危険度が1上がる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2, dangerLevel(dangerDaysPerLevel))
		assert.Equal(t, 3, dangerLevel(dangerDaysPerLevel*2))
	})

	t.Run("日数が増えると危険度は下がらない", func(t *testing.T) {
		t.Parallel()
		assert.Greater(t, dangerLevel(dangerDaysPerLevel*4), dangerLevel(1))
	})

	t.Run("負の入力は最小の1を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, dangerLevel(-5))
	})

	t.Run("同じ入力は常に同じ値", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, dangerLevel(7), dangerLevel(7))
	})
}
