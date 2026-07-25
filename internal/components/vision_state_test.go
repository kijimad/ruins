package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConsumePendingUpdate は視界更新の「要求して消費する」状態機械を固定する。
// private フィールドを RequestUpdate で上げ ConsumePendingUpdate で下げる契約を検証する。
func TestConsumePendingUpdate(t *testing.T) {
	t.Parallel()

	t.Run("要求があれば消費して true を返し下げる", func(t *testing.T) {
		t.Parallel()
		vs := NewVisionState()
		vs.RequestUpdate()

		assert.True(t, vs.ConsumePendingUpdate(), "要求が立っていれば消費で true")
		assert.False(t, vs.ConsumePendingUpdate(), "消費後は下がっていて false")
	})

	t.Run("要求がなければ false を返す", func(t *testing.T) {
		t.Parallel()
		assert.False(t, NewVisionState().ConsumePendingUpdate(), "要求が無ければ false")
	})
}
