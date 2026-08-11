package overlay

import (
	"image"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDetail_初期状態は非表示(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) { return DetailContent{}, false })

	assert.False(t, d.Active())
}

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

func TestDetailOpen_開くと先頭ページに戻る(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		return DetailContent{Name: "回復薬"}, true
	})
	d.page = 3

	d.Open(w.World{})

	assert.True(t, d.Active())
	assert.Equal(t, 0, d.page, "開くと先頭ページに戻る")
}

func TestDetailHandleInput_非表示のときは何もしない(t *testing.T) {
	t.Parallel()
	called := false
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		called = true
		return DetailContent{}, false
	})

	err := d.HandleInput(w.World{})

	require.NoError(t, err)
	assert.False(t, called, "非表示のときprovideは呼ばれない")
	assert.False(t, d.Active())
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

func TestDetailWindow_対象が無ければnilを返す(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) { return DetailContent{}, false })

	got := d.Window(w.World{}, image.Rect(0, 0, 100, 100))

	assert.Nil(t, got)
}

func TestDetailWindow_対象があれば名前とページ位置を表示する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.UIResources = vrt.SharedUIResources(t)
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		return DetailContent{Name: "回復薬", Rows: []entityspec.SpecRow{{Label: "効果", Value: "10"}}}, true
	})
	d.Open(world)

	// widget 生成は ebitenui のグローバル状態に触れるのでロックで直列化する
	var win *widget.Window
	vrt.WithUILock(func() { win = d.Window(world, image.Rect(0, 0, 400, 400)) })

	require.NotNil(t, win)
	labels := collectLabels(win.Contents)
	assert.Contains(t, labels, "回復薬")
	assert.Contains(t, labels, "1/1")
}
