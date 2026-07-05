package tabmenu_test

import (
	"os"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
)

func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

func TestGolden_SingleItem(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	vrt.AssertGolden(t, func() *widget.Container {
		view := tabmenu.NewView(
			tabmenu.Config{
				Tabs: []tabmenu.TabItem{
					{ID: "tab", Label: "タブ", Items: []tabmenu.Item{
						{ID: "item1", Label: "アイテム1"},
					}},
				},
			},
			world,
		)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})
		return view.BuildUI()
	}, 300, 50)
}

func TestGolden_MultipleItems_FirstSelected(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	vrt.AssertGolden(t, func() *widget.Container {
		view := tabmenu.NewView(
			tabmenu.Config{
				Tabs: []tabmenu.TabItem{
					{ID: "tab", Label: "タブ", Items: []tabmenu.Item{
						{ID: "item1", Label: "回復薬"},
						{ID: "item2", Label: "鉄鉱石"},
						{ID: "item3", Label: "聖水"},
					}},
				},
			},
			world,
		)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})
		return view.BuildUI()
	}, 300, 120)
}

func TestGolden_MultipleItems_MiddleSelected(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	vrt.AssertGolden(t, func() *widget.Container {
		view := tabmenu.NewView(
			tabmenu.Config{
				Tabs: []tabmenu.TabItem{
					{ID: "tab", Label: "タブ", Items: []tabmenu.Item{
						{ID: "item1", Label: "回復薬"},
						{ID: "item2", Label: "鉄鉱石"},
						{ID: "item3", Label: "聖水"},
						{ID: "item4", Label: "毒消し"},
						{ID: "item5", Label: "火炎瓶"},
					}},
				},
			},
			world,
		)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 2})
		return view.BuildUI()
	}, 300, 180)
}

func TestGolden_EmptyItems(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	vrt.AssertGolden(t, func() *widget.Container {
		view := tabmenu.NewView(
			tabmenu.Config{
				Tabs: []tabmenu.TabItem{
					{ID: "tab", Label: "タブ", Items: []tabmenu.Item{}},
				},
			},
			world,
		)
		return view.BuildUI()
	}, 300, 50)
}

func TestGolden_WithPagination(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	vrt.AssertGolden(t, func() *widget.Container {
		items := make([]tabmenu.Item, 10)
		for i := range items {
			items[i] = tabmenu.Item{
				ID:    "item",
				Label: "アイテム" + string(rune('A'+i)),
			}
		}
		view := tabmenu.NewView(
			tabmenu.Config{
				Tabs: []tabmenu.TabItem{
					{ID: "tab", Label: "タブ", Items: items},
				},
				ItemsPerPage: 3,
			},
			world,
		)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})
		return view.BuildUI()
	}, 300, 150)
}

func TestGolden_WithAdditionalLabels(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	vrt.AssertGolden(t, func() *widget.Container {
		view := tabmenu.NewView(
			tabmenu.Config{
				Tabs: []tabmenu.TabItem{
					{ID: "tab", Label: "タブ", Items: []tabmenu.Item{
						{ID: "item1", Label: "回復薬", AdditionalLabels: []string{"x3", "1.5kg"}},
						{ID: "item2", Label: "鉄鉱石", AdditionalLabels: []string{"x12", "6.0kg"}},
						{ID: "item3", Label: "聖水", AdditionalLabels: []string{"x1", "0.5kg"}},
					}},
				},
			},
			world,
		)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 0})
		return view.BuildUI()
	}, 400, 120)
}

func TestGolden_ManyItems_LastPage(t *testing.T) {
	t.Parallel()
	world := vrt.InitVRTWorld(t)
	vrt.AssertGolden(t, func() *widget.Container {
		items := make([]tabmenu.Item, 8)
		for i := range items {
			items[i] = tabmenu.Item{
				ID:    "item",
				Label: "アイテム" + string(rune('A'+i)),
			}
		}
		view := tabmenu.NewView(
			tabmenu.Config{
				Tabs: []tabmenu.TabItem{
					{ID: "tab", Label: "タブ", Items: items},
				},
				ItemsPerPage: 3,
			},
			world,
		)
		view.SetState(tabmenu.ViewState{TabIndex: 0, ItemIndex: 7})
		return view.BuildUI()
	}, 300, 120)
}
