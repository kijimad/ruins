package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// comfortableRange は非公開の純粋計算なので内部テストで直接検証する。
func TestComfortableRange(t *testing.T) {
	t.Parallel()

	t.Run("断熱なしの場合の快適温度範囲", func(t *testing.T) {
		t.Parallel()
		lower, upper := comfortableRange(Insulation{Cold: 0, Heat: 0})
		assert.Equal(t, 11, lower)
		assert.Equal(t, 30, upper)
	})

	t.Run("耐寒ありの場合は下限が下がる", func(t *testing.T) {
		t.Parallel()
		lower, upper := comfortableRange(Insulation{Cold: 10, Heat: 0})
		assert.Equal(t, 1, lower)
		assert.Equal(t, 30, upper)
	})

	t.Run("耐熱ありの場合は上限が上がる", func(t *testing.T) {
		t.Parallel()
		lower, upper := comfortableRange(Insulation{Cold: 0, Heat: 10})
		assert.Equal(t, 11, lower)
		assert.Equal(t, 40, upper)
	})

	t.Run("両方ありの場合", func(t *testing.T) {
		t.Parallel()
		lower, upper := comfortableRange(Insulation{Cold: 15, Heat: 5})
		assert.Equal(t, -4, lower)
		assert.Equal(t, 35, upper)
	})
}
