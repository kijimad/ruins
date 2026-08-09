package menuscreen

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
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

// TestEntityDetailContent_死んだ実体は空を返しpanicしない は、生存していない実体を渡しても
// 性能行を引かずに空の内容を返すことを固定する。ゼロ実体への Get で落ちる回帰を防ぐ。
func TestEntityDetailContent_死んだ実体は空を返しpanicしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	var content DetailContent
	assert.NotPanics(t, func() {
		content = EntityDetailContent(world, ecs.Entity{})
	})
	assert.Empty(t, content.Name, "対象が無いので名前は出さない")
	assert.Nil(t, content.Rows, "対象が無いので性能行は出さない")
}
