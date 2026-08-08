package menuscreen

import (
	"testing"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
)

// TestDetail_Open_中身が無ければ開かない は、アイテムの無いメニューで詳細モーダルを開こうとしても
// 開かないことを固定する。開くと中身の無いモーダルが入力を奪い、ESC 以外で抜けられなくなる回帰を防ぐ。
func TestDetail_Open_中身が無ければ開かない(t *testing.T) {
	t.Parallel()

	hasContent := false
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		return DetailContent{Name: "item"}, hasContent
	})
	var world w.World

	d.Open(world)
	assert.False(t, d.Active(), "中身が無いときは開かない")

	hasContent = true
	d.Open(world)
	assert.True(t, d.Active(), "中身があれば開く")
}
