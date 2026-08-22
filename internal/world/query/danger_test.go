package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDangerLevel(t *testing.T) {
	t.Parallel()

	t.Run("危険度は1始まりで最小は1", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, DangerLevel(0))
		assert.Equal(t, 1, DangerLevel(dangerDaysPerLevel-1))
	})

	t.Run("1段ぶんの日数で危険度が1上がる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2, DangerLevel(dangerDaysPerLevel))
		assert.Equal(t, 3, DangerLevel(dangerDaysPerLevel*2))
	})

	t.Run("日数が増えると危険度は下がらない", func(t *testing.T) {
		t.Parallel()
		assert.Greater(t, DangerLevel(dangerDaysPerLevel*4), DangerLevel(1))
	})

	t.Run("負の入力は最小の1を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 1, DangerLevel(-5))
	})

	t.Run("同じ入力は常に同じ値", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, DangerLevel(7), DangerLevel(7))
	})
}
