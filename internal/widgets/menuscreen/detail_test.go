package menuscreen

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
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

// TestDetailContent_resolveRows_NameDescのみは行を出さずpanicしない は、Name と Desc だけを
// 渡した詳細で性能行を解決してもゼロ実体への Get で落ちないことを固定する。人身売買の隊員候補が実例。
func TestDetailContent_resolveRows_NameDescのみは行を出さずpanicしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	c := DetailContent{Name: "Someone", Desc: "Vit5 Str5"} // Entity はゼロ値

	var rows []SpecRow
	assert.NotPanics(t, func() {
		rows = c.resolveRows(world)
	})
	assert.Nil(t, rows, "対象が無いので性能行は出さない")
}
