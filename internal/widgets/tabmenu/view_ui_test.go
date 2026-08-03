package tabmenu_test

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestView_UpdateTabDisplayContainer(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)

	t.Run("ページネーションありならページインジケーターを先頭に加えた件数になる", func(t *testing.T) {
		t.Parallel()
		view := tabmenu.NewView(tabmenu.Config{
			Tabs: []tabmenu.TabItem{
				{ID: "tab", Items: []tabmenu.Item{
					{ID: "i1", Label: "A"},
					{ID: "i2", Label: "B"},
					{ID: "i3", Label: "C"},
					{ID: "i4", Label: "D"},
				}},
			},
			ItemsPerPage: 2,
		}, world)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})

		container := widget.NewContainer()
		view.UpdateTabDisplayContainer(container)

		// ページインジケーター1件 + 表示アイテム2件
		assert.Len(t, container.Children(), 3)
	})

	t.Run("ページネーションなしならページインジケーターを加えない件数になる", func(t *testing.T) {
		t.Parallel()
		view := tabmenu.NewView(tabmenu.Config{
			Tabs: []tabmenu.TabItem{
				{ID: "tab", Items: []tabmenu.Item{
					{ID: "i1", Label: "A"},
					{ID: "i2", Label: "B"},
				}},
			},
		}, world)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})

		container := widget.NewContainer()
		view.UpdateTabDisplayContainer(container)

		assert.Len(t, container.Children(), 2)
	})

	t.Run("アイテムが空なら空表示のテキストを1件加える", func(t *testing.T) {
		t.Parallel()
		view := tabmenu.NewView(tabmenu.Config{
			Tabs: []tabmenu.TabItem{
				{ID: "tab", Items: []tabmenu.Item{}},
			},
		}, world)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})

		container := widget.NewContainer()
		view.UpdateTabDisplayContainer(container)

		require.Len(t, container.Children(), 1)
		_, ok := container.Children()[0].(*widget.Text)
		assert.True(t, ok)
	})

	t.Run("呼び出すたびに既存の子を入れ替えて累積しない", func(t *testing.T) {
		t.Parallel()
		view := tabmenu.NewView(tabmenu.Config{
			Tabs: []tabmenu.TabItem{
				{ID: "tab", Items: []tabmenu.Item{{ID: "i1", Label: "A"}}},
			},
		}, world)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})

		container := widget.NewContainer()
		view.UpdateTabDisplayContainer(container)
		view.UpdateTabDisplayContainer(container)

		assert.Len(t, container.Children(), 1)
	})
}

func TestView_UpdateFocus(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)

	t.Run("BuildUI前に呼んでもパニックしない", func(t *testing.T) {
		t.Parallel()
		view := tabmenu.NewView(tabmenu.Config{
			Tabs: []tabmenu.TabItem{{ID: "tab", Items: []tabmenu.Item{{ID: "i1", Label: "A"}}}},
		}, world)

		assert.NotPanics(t, view.UpdateFocus)
	})

	t.Run("フォーカス移動後も子要素の構成は変わらない", func(t *testing.T) {
		t.Parallel()
		view := tabmenu.NewView(tabmenu.Config{
			Tabs: []tabmenu.TabItem{
				{ID: "tab", Items: []tabmenu.Item{
					{ID: "i1", Label: "A"},
					{ID: "i2", Label: "B"},
					{ID: "i3", Label: "C"},
				}},
			},
		}, world)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})
		container := view.BuildUI()
		before := len(container.Children())

		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 2})
		require.NotPanics(t, view.UpdateFocus)

		assert.Len(t, container.Children(), before)
	})
}
