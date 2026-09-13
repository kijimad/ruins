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

// latitudeColdForDepth は非公開の純粋計算なので内部テストで直接検証する。
// 具体値でなく勾配の形、すなわち手前0・単調増加・頭打ちを固定する。
func TestLatitudeColdForDepth(t *testing.T) {
	t.Parallel()

	t.Run("手前と負の距離では寒くならない", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, latitudeColdForDepth(0), "起点は勾配なし")
		assert.Equal(t, 0, latitudeColdForDepth(-5), "負の距離も0")
	})

	t.Run("奥へ進むほど単調に寒くなる", func(t *testing.T) {
		t.Parallel()
		prev := latitudeColdForDepth(0)
		for d := 1; d <= latitudeColdMax/latitudeColdPerChunk; d++ {
			cur := latitudeColdForDepth(d)
			assert.GreaterOrEqual(t, cur, prev, "距離が増えれば寒さは減らない")
			prev = cur
		}
	})

	t.Run("十分奥では頭打ちになる", func(t *testing.T) {
		t.Parallel()
		depthCap := latitudeColdMax / latitudeColdPerChunk
		assert.Equal(t, latitudeColdMax, latitudeColdForDepth(depthCap), "上限に到達する")
		assert.Equal(t, latitudeColdMax, latitudeColdForDepth(depthCap*10), "上限を超えて寒くならない")
	})
}
