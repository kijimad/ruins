package menuscreen

import (
	"fmt"
	"image"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDetail_初期状態は非表示(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) { return DetailContent{}, false })

	assert.False(t, d.Active())
}

func TestDetailOpen_先頭ページで表示状態にする(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) { return DetailContent{}, false })
	d.page = 3

	d.Open()

	assert.True(t, d.Active())
	assert.Equal(t, 0, d.page)
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

func TestDetailContentResolveRows_Rowsが指定されていれば優先する(t *testing.T) {
	t.Parallel()
	explicit := []views.SpecRow{{Label: "任意", Value: "行"}}
	c := DetailContent{
		Rows: explicit,
		Spec: &gc.EntitySpec{Value: &gc.Value{Value: 999}},
	}

	got := c.resolveRows(w.World{})

	assert.Equal(t, explicit, got)
}

func TestDetailContentResolveRows_RowsがなければSpecから解決する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	c := DetailContent{Spec: &gc.EntitySpec{Value: &gc.Value{Value: 1200}}}

	got := c.resolveRows(world)

	assert.Equal(t, []views.SpecRow{{Label: "価値", Value: query.FormatCurrency(1200)}}, got)
}

func TestDetailContentResolveRows_RowsもSpecも無ければEntityから解決する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()
	world.Components.Value.Add(e, &gc.Value{Value: 500})
	c := DetailContent{Entity: e}

	got := c.resolveRows(world)

	assert.Equal(t, []views.SpecRow{{Label: "価値", Value: query.FormatCurrency(500)}}, got)
}

func TestDetailWindow_対象が無ければnilを返す(t *testing.T) {
	t.Parallel()
	d := NewDetail(func(_ w.World) (DetailContent, bool) { return DetailContent{}, false })

	got := d.Window(w.World{}, image.Rect(0, 0, 100, 100))

	assert.Nil(t, got)
}

//nolint:paralleltest // ebitenui内部のrace conditionのためt.Parallel()を使用しない
func TestDetailWindow_対象があれば名前とページ位置を表示する(t *testing.T) {
	world := testutil.InitTestWorld(t)
	world.Resources.UIResources = vrt.SharedUIResources(t)
	d := NewDetail(func(_ w.World) (DetailContent, bool) {
		return DetailContent{Name: "回復薬", Rows: []views.SpecRow{{Label: "効果", Value: "10"}}}, true
	})
	d.Open()

	win := d.Window(world, image.Rect(0, 0, 400, 400))

	require.NotNil(t, win)
	labels := collectLabels(win.Contents)
	assert.Contains(t, labels, "回復薬")
	assert.Contains(t, labels, fmt.Sprintf("%s 1/1 %s", consts.IconArrowLeft, consts.IconArrowRight))
}
