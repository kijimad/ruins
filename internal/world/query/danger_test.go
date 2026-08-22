package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDangerLevel(t *testing.T) {
	t.Parallel()

	t.Run("日数が増えると危険度は下がらない", func(t *testing.T) {
		t.Parallel()
		early := DangerLevel(1)
		late := DangerLevel(dangerDaysPerLevel * 4)
		assert.Greater(t, late, early, "日数が進めば危険度は上がるはず")
	})

	t.Run("1段ぶんの日数で危険度が1上がる", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, DangerLevel(dangerDaysPerLevel-1))
		assert.Equal(t, 1, DangerLevel(dangerDaysPerLevel))
	})

	t.Run("負の入力は0として扱う", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, DangerLevel(-5))
	})

	t.Run("同じ入力は常に同じ値", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, DangerLevel(7), DangerLevel(7))
	})
}
