package uicore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStretchNeedsBlend(t *testing.T) {
	t.Parallel()

	t.Run("等倍なら混ぜない", func(t *testing.T) {
		t.Parallel()
		assert.False(t, stretchNeedsBlend(1, 10))
	})

	t.Run("1テクセルしか無い軸は混ぜない", func(t *testing.T) {
		t.Parallel()
		assert.False(t, stretchNeedsBlend(2, 1))
	})

	t.Run("縮小方向で複数テクセルなら混ぜる", func(t *testing.T) {
		t.Parallel()
		assert.True(t, stretchNeedsBlend(0.5, 4))
	})

	t.Run("拡大方向で複数テクセルなら混ぜる", func(t *testing.T) {
		t.Parallel()
		assert.True(t, stretchNeedsBlend(2, 4))
	})
}
